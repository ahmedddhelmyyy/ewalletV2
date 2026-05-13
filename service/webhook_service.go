package service

// import (
// 	"context"
// 	"errors"
// 	"fmt"

// 	"github.com/google/uuid"
// 	"github.com/ewallet/model"
// 	"github.com/ewallet/repository"
// )

// var ErrDuplicateWebhook = errors.New("webhook: transaction already processed")

// type WebhookService interface {
// 	ProcessIncomingTransfer(ctx context.Context, walletID uuid.UUID, payload model.IncomingTransferPayload) error
// }

// type webhookService struct {
// 	txRepo     repository.TransactionRepository
// 	walletRepo repository.WalletRepository
// }

// func NewWebhookService(txRepo repository.TransactionRepository, walletRepo repository.WalletRepository) WebhookService {
// 	return &webhookService{txRepo: txRepo, walletRepo: walletRepo}
// }

// func (s *webhookService) ProcessIncomingTransfer(ctx context.Context, walletID uuid.UUID, payload model.IncomingTransferPayload) error {
// 	// 1. Idempotency check — reject if we've seen this external_id before
// 	exists, err := s.txRepo.ExistsByExternalID(ctx, payload.ExternalID)
// 	if err != nil {
// 		return fmt.Errorf("webhook idempotency check: %w", err)
// 	}
// 	if exists {
// 		return ErrDuplicateWebhook
// 	}

// 	// 2. Load the recipient wallet (ours) to get its wallet number
// 	wallet, err := s.walletRepo.FindByID(ctx, walletID)
// 	if err != nil {
// 		return fmt.Errorf("webhook wallet lookup: %w", err)
// 	}

// 	// 3. Map payload → Transaction
// 	externalID := payload.ExternalID
// 	tx := &model.Transaction{
// 		RecipientWalletID:     &walletID,
// 		RecipientWalletNumber: &wallet.WalletNumber,
// 		SenderWalletNumber:    &payload.SenderNumber,
// 		CounterpartName:       payload.SenderName,
// 		ExternalID:            &externalID,
// 		Type:                  "receive",
// 		Status:                "completed",
// 		Amount:                payload.Amount,
// 		Currency:              payload.Currency,
// 		Category:              "transfer",
// 		Note:                  payload.Note,
// 	}

// 	// 4. Persist + credit the wallet in one DB transaction
// 	if err := s.txRepo.CreateWithBalanceCredit(ctx, tx, walletID, payload.Amount); err != nil {
// 		return fmt.Errorf("webhook persist: %w", err)
// 	}

// 	return nil
// }