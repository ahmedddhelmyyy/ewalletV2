package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"gorm.io/gorm"
)

// UserRepository defines all database operations for the User model.
type UserRepository interface {
	WithTx(tx *gorm.DB) UserRepository
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
}

// WalletRepository defines all database operations for the Wallet model.
type WalletRepository interface {
	WithTx(tx *gorm.DB) WalletRepository
	Create(wallet *model.Wallet) error
	FindByUserID(userID uuid.UUID) (*model.Wallet, error)
	FindByWalletNumber(walletNumber string) (*model.Wallet, error)
	UpdateBalance(walletID uuid.UUID, newBalance int64) error
	Count() (int64, error)
}

// TransactionRepository defines all database operations for the Transaction model.
type TransactionRepository interface {
	WithTx(tx *gorm.DB) TransactionRepository
	Create(tx *model.Transaction) error
	FindByID(id uuid.UUID) (*model.Transaction, error)
	FindByWallet(walletID uuid.UUID, filters model.TransactionFilters) ([]model.Transaction, int64, error)
	GetCategorySummary(walletID uuid.UUID, from, to time.Time) ([]CategorySummaryRow, error)
	GetDailyInFlow(walletID uuid.UUID, from, to time.Time) ([]DailyFlowRow, error)
	GetDailyOutFlow(walletID uuid.UUID, from, to time.Time) ([]DailyFlowRow, error)
}

// BillRepository defines all database operations for the Bill model.
type BillRepository interface {
	WithTx(tx *gorm.DB) BillRepository
	Create(bill *model.Bill) error
	FindByID(id uuid.UUID) (*model.Bill, error)
	FindByUserID(userID uuid.UUID, filters model.BillFilters) ([]model.Bill, int64, error)
	Save(bill *model.Bill) error
	Delete(id uuid.UUID) error
}

// RefreshTokenRepository defines all database operations for the RefreshToken model.
type RefreshTokenRepository interface {
	Create(token *model.RefreshToken) error
	FindByToken(token string) (*model.RefreshToken, error)
	MarkUsed(id uuid.UUID) error
	DeleteByUserID(userID uuid.UUID) error
}

// CategorySummaryRow is returned by the expense category aggregation query.
type CategorySummaryRow struct {
	Category string
	Total    int64
	TxCount  int64
}

// DailyFlowRow is returned by the daily in/out aggregation queries.
type DailyFlowRow struct {
	Date   string
	Amount int64
}
