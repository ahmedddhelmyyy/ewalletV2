package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"gorm.io/gorm"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrTransactionNotFound = errors.New("transaction not found")
var ErrTransactionExpired = errors.New("transaction expired")
var ErrAlreadyProcessed = errors.New("transaction already processed")

type MerchantService struct {
	db          *gorm.DB
	walletSvc   MerchantWalletService
	webhookSvc  WebhookDispatcher
}

type MerchantWalletService interface {
	GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error)
	Deduct(ctx context.Context, userID string, amount int64) error
	CreditMerchant(ctx context.Context, merchantID string, amount int64) error
	GetMerchantBalance(ctx context.Context, merchantID string) (int64, error)
}

type WebhookDispatcher interface {
	Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error
}

func NewMerchantService(db *gorm.DB, walletSvc MerchantWalletService, webhookSvc WebhookDispatcher) *MerchantService {
	return &MerchantService{db: db, walletSvc: walletSvc, webhookSvc: webhookSvc}
}

func (s *MerchantService) CreateTransaction(ctx context.Context, merchantID string, merchantUserID string, req *model.CreateMerchantTransactionRequest, idempotencyKey string) (*model.MerchantTransaction, bool, error) {
	if idempotencyKey != "" {
		var existing model.MerchantTransaction
		err := s.db.WithContext(ctx).Where("idempotency_key = ? AND merchant_id = ?", idempotencyKey, merchantID).First(&existing).Error
		if err == nil {
			if existing.MerchantID != merchantID {
				return nil, false, fmt.Errorf("idempotency key already exists for a different merchant")
			}
			return &existing, false, nil
		}
	}

	tx := &model.MerchantTransaction{
		MerchantID:     merchantID,
		MerchantUserID: merchantUserID,
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         "pending",
		ReturnURL:      req.ReturnURL,
		CancelURL:      req.CancelURL,
		WebhookURL:     req.WebhookURL,
		RedirectToken:  uuid.New().String(),
		ExpiresAt:      time.Now().Add(15 * time.Minute),
	}
	if idempotencyKey != "" {
		tx.IdempotencyKey = idempotencyKey
	}

	if err := s.db.WithContext(ctx).Create(tx).Error; err != nil {
		return nil, false, err
	}

	return tx, true, nil
}

func (s *MerchantService) GetTransactionByID(ctx context.Context, txID string, merchantID string) (*model.MerchantTransaction, error) {
	var tx model.MerchantTransaction
	if err := s.db.WithContext(ctx).Where("id = ? AND merchant_id = ?", txID, merchantID).First(&tx).Error; err != nil {
		return nil, ErrTransactionNotFound
	}
	return &tx, nil
}

func (s *MerchantService) GetTransactionByToken(ctx context.Context, token string) (*model.MerchantTransaction, error) {
	var tx model.MerchantTransaction
	if err := s.db.WithContext(ctx).Where("redirect_token = ?", token).First(&tx).Error; err != nil {
		return nil, ErrTransactionNotFound
	}
	return &tx, nil
}

func (s *MerchantService) ConfirmTransaction(ctx context.Context, token string, userUUID uuid.UUID) (*model.MerchantTransaction, error) {
	tx, err := s.GetTransactionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if tx.Status != "pending" {
		return nil, ErrAlreadyProcessed
	}

	if time.Now().After(tx.ExpiresAt) {
		tx.Status = "expired"
		s.db.WithContext(ctx).Save(tx)
		return nil, ErrTransactionExpired
	}

	var customerWallet model.Wallet
	if err := s.db.WithContext(ctx).Where("user_id = ?", userUUID).Preload("User").First(&customerWallet).Error; err != nil {
		return nil, fmt.Errorf("customer wallet not found: %w", err)
	}

	if customerWallet.Balance < tx.Amount {
		tx.Status = "failed"
		s.db.WithContext(ctx).Save(tx)
		return nil, ErrInsufficientBalance
	}

	merchantUserUUID, err := uuid.Parse(tx.MerchantUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid merchant user id: %w", err)
	}

	var merchantWallet model.Wallet
	if err := s.db.WithContext(ctx).Where("user_id = ?", merchantUserUUID).Preload("User").First(&merchantWallet).Error; err != nil {
		return nil, fmt.Errorf("merchant wallet not found: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		if err := dbTx.Model(&customerWallet).Update("balance", customerWallet.Balance-tx.Amount).Error; err != nil {
			return err
		}
		if err := dbTx.Model(&merchantWallet).Update("balance", merchantWallet.Balance+tx.Amount).Error; err != nil {
			return err
		}

		debitTx := &model.Transaction{
			SenderWalletID:        &customerWallet.ID,
			SenderWalletNumber:    &customerWallet.WalletNumber,
			RecipientWalletID:     &merchantWallet.ID,
			RecipientWalletNumber: &merchantWallet.WalletNumber,
			Type:                  "send",
			Status:                "completed",
			Amount:                tx.Amount,
			Currency:              tx.Currency,
			Category:              "transfer",
		}
		if err := dbTx.Create(debitTx).Error; err != nil {
			return err
		}

		creditTx := &model.Transaction{
			SenderWalletID:        &customerWallet.ID,
			SenderWalletNumber:    &customerWallet.WalletNumber,
			RecipientWalletID:     &merchantWallet.ID,
			RecipientWalletNumber: &merchantWallet.WalletNumber,
			Type:                  "receive",
			Status:                "completed",
			Amount:                tx.Amount,
			Currency:              tx.Currency,
			Category:              "transfer",
		}
		if err := dbTx.Create(creditTx).Error; err != nil {
			return err
		}

		tx.Status = "success"
		tx.UserID = userUUID.String()
		return dbTx.Save(tx).Error
	})

	if err != nil {
		tx.Status = "failed"
		s.db.WithContext(ctx).Save(tx)
		go func(whURL, whSecret string, txn *model.MerchantTransaction) {
			whURL = tx.WebhookURL
			if whURL == "" {
				return
			}
			s.webhookSvc.Dispatch(context.Background(), model.Merchant{WebhookURL: whURL, WebhookSecret: whSecret}, "transaction.failed", txn)
		}(tx.WebhookURL, "whsec_test", tx)
		return nil, err
	}

	go func(whURL, whSecret string, txn *model.MerchantTransaction) {
		whURL = tx.WebhookURL
		if whURL == "" {
			return
		}
		s.webhookSvc.Dispatch(context.Background(), model.Merchant{WebhookURL: whURL, WebhookSecret: whSecret}, "transaction.success", txn)
	}(tx.WebhookURL, "whsec_test", tx)

	return tx, nil
}

func (s *MerchantService) ExpireTransaction(ctx context.Context, txID string) error {
	return s.db.WithContext(ctx).Model(&model.MerchantTransaction{}).Where("id = ? AND status = ?", txID, "pending").Update("status", "expired").Error
}

func (s *MerchantService) GetMerchantBalance(ctx context.Context, merchantID string) (int64, error) {
	return s.walletSvc.GetMerchantBalance(ctx, merchantID)
}