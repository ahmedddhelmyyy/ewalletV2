package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"gorm.io/gorm"
)

type walletRepository struct {
	db *gorm.DB
}

// NewWalletRepository creates a new WalletRepository backed by the given *gorm.DB.
func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) WithTx(tx *gorm.DB) WalletRepository {
	return &walletRepository{db: tx}
}

func (r *walletRepository) Create(wallet *model.Wallet) error {
	return r.db.Create(wallet).Error
}

func (r *walletRepository) FindByUserID(userID uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	err := r.db.Preload("User").Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrWalletNotFound
		}
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) FindByWalletNumber(walletNumber string) (*model.Wallet, error) {
	var wallet model.Wallet
	err := r.db.Preload("User").Where("wallet_number = ?", walletNumber).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrWalletNotFound
		}
		return nil, err
	}
	return &wallet, nil
}

// UpdateBalance sets the wallet balance to newBalance.
// This should always be called inside a database transaction for atomicity.
func (r *walletRepository) UpdateBalance(walletID uuid.UUID, newBalance int64) error {
	result := r.db.Model(&model.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", newBalance)
	return result.Error
}

// Count returns the total number of wallets in the system (used for wallet number generation).
func (r *walletRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Wallet{}).Count(&count).Error
	return count, err
}
