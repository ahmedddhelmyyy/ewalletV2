package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ewallet/model"
)

// AuthService handles user registration, login, logout and token refresh.
type AuthService interface {
	Register(req model.RegisterRequest) (*model.RegisterResponse, error)
	Login(req model.LoginRequest) (*model.LoginResponse, error)
	Logout(userID uuid.UUID, refreshToken string) error
	RefreshToken(refreshToken string) (*model.RefreshTokenResponse, error)
}

// WalletService handles wallet retrieval.
type WalletService interface {
	GetWalletByUserID(userID uuid.UUID) (*model.WalletResponse, error)
}

// TransactionService handles all money movement operations.
type TransactionService interface {
	Send(userID uuid.UUID, req model.SendRequest) (*model.SendResponse, error)
	TopUp(userID uuid.UUID, req model.TopUpRequest) (*model.TopUpResponse, error)
	Withdraw(userID uuid.UUID, req model.WithdrawRequest) (*model.WithdrawResponse, error)
	GetHistory(userID uuid.UUID, filters model.TransactionFilters) ([]model.TransactionResponse, int64, error)
	GetByID(userID uuid.UUID, transactionID uuid.UUID) (*model.TransactionResponse, error)
}

// BillService handles bill lifecycle operations.
type BillService interface {
	Create(userID uuid.UUID, req model.CreateBillRequest) (*model.BillResponse, error)
	List(userID uuid.UUID, filters model.BillFilters) ([]model.BillResponse, int64, error)
	GetByID(userID uuid.UUID, billID uuid.UUID) (*model.BillResponse, error)
	Pay(userID uuid.UUID, billID uuid.UUID) (*model.PayBillResponse, error)
	Delete(userID uuid.UUID, billID uuid.UUID) error
}

// ExpenseService handles expense analytics queries.
type ExpenseService interface {
	GetSummary(userID uuid.UUID, from, to time.Time) (*model.ExpenseSummaryResponse, error)
	GetFlow(userID uuid.UUID, from, to time.Time) (*model.MoneyFlowResponse, error)
}

// MerchantServiceInterface handles merchant payment transaction operations.
type MerchantServiceInterface interface {
	CreateTransaction(ctx context.Context, merchantID string, merchantUserID string, req *model.CreateMerchantTransactionRequest, idempotencyKey string) (*model.MerchantTransaction, bool, error)
	GetTransactionByID(ctx context.Context, txID string, merchantID string) (*model.MerchantTransaction, error)
	GetTransactionByToken(ctx context.Context, token string) (*model.MerchantTransaction, error)
	ConfirmTransaction(ctx context.Context, token string, userUUID uuid.UUID) (*model.MerchantTransaction, error)
	ExpireTransaction(ctx context.Context, txID string) error
	GetMerchantBalance(ctx context.Context, merchantID string) (int64, error)
}

// MerchantBalanceService handles merchant balance queries.
type MerchantBalanceService interface {
	GetMerchantBalance(ctx context.Context, merchantID string) (int64, error)
}
