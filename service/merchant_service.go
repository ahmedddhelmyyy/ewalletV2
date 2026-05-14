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
}

type WebhookDispatcher interface {
	Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error
}

func NewMerchantService(db *gorm.DB, walletSvc MerchantWalletService, webhookSvc WebhookDispatcher) *MerchantService {
	return &MerchantService{db: db, walletSvc: walletSvc, webhookSvc: webhookSvc}
}

func (s *MerchantService) CreateTransaction(ctx context.Context, merchantID string, req *model.CreateMerchantTransactionRequest, idempotencyKey string) (*model.MerchantTransaction, bool, error) {
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
		OrderID:        req.OrderID,
		IdempotencyKey: idempotencyKey,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         "pending",
		ReturnURL:      req.ReturnURL,
		CancelURL:      req.CancelURL,
		RedirectToken:  uuid.New().String(),
		ExpiresAt:      time.Now().Add(15 * time.Minute),
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

func (s *MerchantService) ConfirmTransaction(ctx context.Context, token string, userID string) (*model.MerchantTransaction, error) {
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

	if err := s.walletSvc.Deduct(ctx, userID, tx.Amount); err != nil {
		tx.Status = "failed"
		s.db.WithContext(ctx).Save(tx)
		go func() {
			merchant := model.Merchant{
				ID:            tx.MerchantID,
				WebhookURL:    "https://shop.com/api/payments/wallet/webhook",
				WebhookSecret: "whsec_test",
			}
			s.webhookSvc.Dispatch(context.Background(), merchant, "transaction.failed", tx)
		}()
		return nil, ErrInsufficientBalance
	}

	tx.Status = "success"
	tx.UserID = userID
	s.db.WithContext(ctx).Save(tx)

	s.walletSvc.CreditMerchant(ctx, tx.MerchantID, tx.Amount)

	go func() {
		merchant := model.Merchant{
			ID:            tx.MerchantID,
			WebhookURL:    "https://shop.com/api/payments/wallet/webhook",
			WebhookSecret: "whsec_test",
		}
		s.webhookSvc.Dispatch(context.Background(), merchant, "transaction.success", tx)
	}()

	return tx, nil
}

func (s *MerchantService) ExpireTransaction(ctx context.Context, txID string) error {
	return s.db.WithContext(ctx).Model(&model.MerchantTransaction{}).Where("id = ? AND status = ?", txID, "pending").Update("status", "expired").Error
}