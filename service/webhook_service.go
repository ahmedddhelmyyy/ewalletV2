package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

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

type webhookPayload struct {
	Event         string `json:"event"`
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	UserID        string `json:"user_id"`
	Timestamp     string `json:"timestamp"`
}

func (s *webhookService) Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error {
	if merchant.WebhookURL == "" {
		return nil
	}

	payload := webhookPayload{
		Event:         event,
		TransactionID: tx.ID.String(),
		OrderID:       tx.OrderID,
		Status:        tx.Status,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		UserID:        tx.UserID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook marshal: %w", err)
	}

	var sig string
	if merchant.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(merchant.WebhookSecret))
		mac.Write(body)
		sig = hex.EncodeToString(mac.Sum(nil))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, merchant.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", sig)
	req.Header.Set("X-Webhook-Event", event)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WEBHOOK] failed to send webhook to %s: %v", merchant.WebhookURL, err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("[WEBHOOK] sent %s to %s — status %d", event, merchant.WebhookURL, resp.StatusCode)
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
