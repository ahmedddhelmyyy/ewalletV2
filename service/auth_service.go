///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/service/auth_service.go
package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ewallet/config"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// JWTClaims are the custom claims embedded in the access token.
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type authService struct {
	db          *gorm.DB
	cfg         *config.Config
	userRepo    repository.UserRepository
	walletRepo  repository.WalletRepository
	tokenRepo   repository.RefreshTokenRepository
}

// NewAuthService creates an AuthService with all required dependencies.
func NewAuthService(
	db *gorm.DB,
	cfg *config.Config,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	tokenRepo repository.RefreshTokenRepository,
) AuthService {
	return &authService{
		db:         db,
		cfg:        cfg,
		userRepo:   userRepo,
		walletRepo: walletRepo,
		tokenRepo:  tokenRepo,
	}
}

// Register creates a new user and provisions their wallet atomically.
func (s *authService) Register(req model.RegisterRequest) (*model.RegisterResponse, error) {
	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt: %w", err)
	}

	var (
		newUser   model.User
		newWallet model.Wallet
	)

	// Use a DB transaction so user + wallet are created atomically
	err = s.db.Transaction(func(tx *gorm.DB) error {
		userRepo := s.userRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		newUser = model.User{
			FullName: req.FullName,
			Email:    req.Email,
			Password: string(hashed),
		}
		if err := userRepo.Create(&newUser); err != nil {
			return err
		}

		// Generate a unique wallet number using current count
		count, err := walletRepo.Count()
		if err != nil {
			return fmt.Errorf("wallet count: %w", err)
		}
		walletNumber := fmt.Sprintf("WAL-%s-%05d", time.Now().UTC().Format("20060102"), count+1)

		newWallet = model.Wallet{
			UserID:       newUser.ID,
			WalletNumber: walletNumber,
			Balance:      0,
			Currency:     model.DefaultCurrency,
		}
		return walletRepo.Create(&newWallet)
	})

	if err != nil {
		// Translate GORM unique constraint violation to a domain error
		if isDuplicateKeyError(err) {
			return nil, model.ErrEmailAlreadyExists
		}
		return nil, err
	}

	// Issue tokens
	tokens, err := s.issueTokenPair(newUser.ID)
	if err != nil {
		return nil, err
	}

	return &model.RegisterResponse{
		User:   mapUserToResponse(&newUser),
		Wallet: mapWalletToResponse(&newWallet, nil),
		Tokens: *tokens,
	}, nil
}

// Login validates credentials and returns a token pair.
func (s *authService) Login(req model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, model.ErrInvalidCredentials
	}

	tokens, err := s.issueTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		User:   mapUserToResponse(user),
		Tokens: *tokens,
	}, nil
}

// Logout invalidates all refresh tokens belonging to the user.
func (s *authService) Logout(userID uuid.UUID, refreshToken string) error {
	// Verify the token exists and belongs to this user before deleting
	rt, err := s.tokenRepo.FindByToken(refreshToken)
	if err != nil {
		return err // ErrTokenNotFound propagates as-is
	}
	if rt.UserID != userID {
		return model.ErrForbidden
	}
	// Delete all tokens for the user — full logout
	return s.tokenRepo.DeleteByUserID(userID)
}

// RefreshToken validates a refresh token, rotates it, and returns a new pair.
func (s *authService) RefreshToken(refreshToken string) (*model.RefreshTokenResponse, error) {
	rt, err := s.tokenRepo.FindByToken(refreshToken)
	if err != nil {
		if errors.Is(err, model.ErrTokenNotFound) {
			return nil, model.ErrInvalidRefreshToken
		}
		return nil, err
	}

	if rt.Used || rt.IsExpired() {
		return nil, model.ErrInvalidRefreshToken
	}

	// Mark current token as used (rotation)
	if err := s.tokenRepo.MarkUsed(rt.ID); err != nil {
		return nil, err
	}

	// Issue a fresh pair
	tokens, err := s.issueTokenPair(rt.UserID)
	if err != nil {
		return nil, err
	}

	return &model.RefreshTokenResponse{Tokens: *tokens}, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// issueTokenPair generates a signed JWT access token and a random refresh token,
// persists the refresh token, and returns both with their expiry times.
func (s *authService) issueTokenPair(userID uuid.UUID) (*model.TokenPair, error) {
	now := time.Now().UTC()
	accessExpiry := now.Add(time.Duration(s.cfg.AccessTokenDurationMinutes) * time.Minute)
	refreshExpiry := now.Add(time.Duration(s.cfg.RefreshTokenDurationDays) * 24 * time.Hour)

	// Sign JWT
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign jwt: %w", err)
	}

	// Generate a cryptographically random refresh token
	rawBytes := make([]byte, 64)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("rand refresh token: %w", err)
	}
	rawRefreshToken := base64.URLEncoding.EncodeToString(rawBytes)

	rt := &model.RefreshToken{
		UserID:    userID,
		Token:     rawRefreshToken,
		ExpiresAt: refreshExpiry,
		Used:      false,
	}
	if err := s.tokenRepo.Create(rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          rawRefreshToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

// isDuplicateKeyError detects database unique-constraint violations.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return contains(errMsg, "duplicate key") ||
		contains(errMsg, "unique constraint") ||
		contains(errMsg, "UNIQUE constraint failed") ||
		contains(errMsg, "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ─── Mapping helpers ──────────────────────────────────────────────────────────

func mapUserToResponse(u *model.User) model.UserResponse {
	return model.UserResponse{
		ID:        u.ID,
		FullName:  u.FullName,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func mapWalletToResponse(w *model.Wallet, owner *model.WalletOwnerResponse) model.WalletResponse {
	return model.WalletResponse{
		ID:           w.ID,
		WalletNumber: w.WalletNumber,
		Balance:      w.Balance,
		Currency:     w.Currency,
		Owner:        owner,
		CreatedAt:    w.CreatedAt,
		UpdatedAt:    w.UpdatedAt,
	}
}
