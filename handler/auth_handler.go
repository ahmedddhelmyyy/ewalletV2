///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/handler/auth_handler.go
package handler

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/ewallet/model"
	"github.com/ewallet/service"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// AuthHandler holds HTTP handlers for authentication endpoints.
type AuthHandler struct {
	authSvc service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Creates a user account and auto-provisions a wallet. Returns JWT tokens immediately so no second login is needed.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		model.RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	SuccessEnvelope{data=model.RegisterResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Validation error"
//	@Failure		409		{object}	ErrorEnvelope	"Email already exists"
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	errs := map[string]string{}
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if len(req.FullName) < 2 || len(req.FullName) > 100 {
		errs["full_name"] = "Must be between 2 and 100 characters."
	}
	if !emailRegex.MatchString(req.Email) {
		errs["email"] = "Must be a valid email address."
	}
	if pwErr := validatePassword(req.Password); pwErr != "" {
		errs["password"] = pwErr
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	resp, err := h.authSvc.Register(req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, resp)
}

// Login godoc
//
//	@Summary		Login
//	@Description	Validates credentials and returns a JWT access token + refresh token pair. Access token TTL: 15 min. Refresh token TTL: 7 days.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		model.LoginRequest	true	"Login credentials"
//	@Success		200		{object}	SuccessEnvelope{data=model.LoginResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Missing fields"
//	@Failure		401		{object}	ErrorEnvelope	"Invalid credentials"
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	errs := map[string]string{}
	if req.Email == "" {
		errs["email"] = "Email is required."
	}
	if req.Password == "" {
		errs["password"] = "Password is required."
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	resp, err := h.authSvc.Login(req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}

// Logout godoc
//
//	@Summary		Logout
//	@Description	Invalidates the refresh token. The access token expires naturally after 15 minutes.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		model.LogoutRequest	true	"Refresh token to invalidate"
//	@Success		200		{object}	SuccessEnvelope
//	@Failure		400		{object}	ErrorEnvelope	"Missing refresh token"
//	@Failure		401		{object}	ErrorEnvelope	"Unauthorized"
//	@Failure		404		{object}	ErrorEnvelope	"Token not found"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		validationError(w, map[string]string{"refresh_token": "Refresh token is required."})
		return
	}

	userID := userIDFromCtx(r)
	if err := h.authSvc.Logout(userID, req.RefreshToken); err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "Logged out successfully."})
}

// RefreshToken godoc
//
//	@Summary		Refresh access token
//	@Description	Exchanges a valid refresh token for a brand-new token pair. The old refresh token is invalidated (rotation enforced).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		model.RefreshTokenRequest	true	"Refresh token"
//	@Success		200		{object}	SuccessEnvelope{data=model.RefreshTokenResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Missing refresh token"
//	@Failure		401		{object}	ErrorEnvelope	"Invalid or expired refresh token"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		validationError(w, map[string]string{"refresh_token": "Refresh token is required."})
		return
	}

	resp, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}

func validatePassword(pw string) string {
	if len(pw) < 8 {
		return "Must be at least 8 characters."
	}
	hasUpper, hasDigit := false, false
	for _, ch := range pw {
		if unicode.IsUpper(ch) {
			hasUpper = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasUpper {
		return "Must contain at least one uppercase letter."
	}
	if !hasDigit {
		return "Must contain at least one digit."
	}
	return ""
}