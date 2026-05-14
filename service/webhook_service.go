package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
)

var ErrDuplicateWebhook = errors.New("webhook: transaction already processed")

type WebhookService interface {
	ProcessIncomingTransfer(ctx context.Context, walletID uuid.UUID, payload model.IncomingTransferPayload) error
	Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error
}

type webhookService struct {
	txRepo     repository.TransactionRepository
	walletRepo repository.WalletRepository
}

func NewWebhookService(txRepo repository.TransactionRepository, walletRepo repository.WalletRepository) WebhookService {
	return &webhookService{txRepo: txRepo, walletRepo: walletRepo}
}

func (s *webhookService) Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error {
	return nil
}

func (s *webhookService) ProcessIncomingTransfer(ctx context.Context, walletID uuid.UUID, payload model.IncomingTransferPayload) error {
	exists, err := s.txRepo.ExistsByExternalID(ctx, payload.ExternalID)
	if err != nil {
		return fmt.Errorf("webhook idempotency check: %w", err)
	}
	if exists {
		return ErrDuplicateWebhook
	}

	wallet, err := s.walletRepo.FindByID(walletID)
	if err != nil {
		return fmt.Errorf("webhook wallet lookup: %w", err)
	}

	externalID := payload.ExternalID
	tx := &model.Transaction{
		RecipientWalletID:     &walletID,
		RecipientWalletNumber: &wallet.WalletNumber,
		SenderWalletNumber:    &payload.SenderNumber,
		CounterpartName:       payload.SenderName,
		ExternalID:            &externalID,
		Type:                  "receive",
		Status:                "completed",
		Amount:                payload.Amount,
		Currency:              payload.Currency,
		Category:              "transfer",
		Note:                  payload.Note,
	}

	if err := s.txRepo.CreateWithBalanceCredit(ctx, tx, walletID, payload.Amount); err != nil {
		return fmt.Errorf("webhook persist: %w", err)
	}

	return nil
}