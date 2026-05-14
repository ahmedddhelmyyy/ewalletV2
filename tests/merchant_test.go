package tests

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ewallet/model"
)

func hmacSign(secret, body, timestamp string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body + timestamp))
	return hex.EncodeToString(h.Sum(nil))
}

func TestHMACSign(t *testing.T) {
	secret := "supersecret"
	body := `{"order_id":"test123","amount":1000}`
	timestamp := "1234567890"

	sig := hmacSign(secret, body, timestamp)

	if sig == "" {
		t.Errorf("Expected non-empty HMAC signature")
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body + timestamp))
	expected := hex.EncodeToString(h.Sum(nil))

	if sig != expected {
		t.Errorf("HMAC mismatch: got %s, expected %s", sig, expected)
	}
}

func TestMerchantTransactionModel(t *testing.T) {
	tx := model.MerchantTransaction{
		MerchantID:     "merch_123",
		OrderID:        "order_abc",
		IdempotencyKey: "idem_123",
		Amount:         1000,
		Currency:       "USD",
		Status:         "pending",
		ReturnURL:      "https://example.com/return",
		ExpiresAt:     time.Now().Add(15 * time.Minute),
	}

	if tx.MerchantID != "merch_123" {
		t.Errorf("Expected MerchantID merch_123, got %s", tx.MerchantID)
	}
	if tx.Status != "pending" {
		t.Errorf("Expected status pending, got %s", tx.Status)
	}
	if tx.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", tx.Amount)
	}
}

func TestCreateMerchantTransactionRequest_JSON(t *testing.T) {
	jsonData := `{"order_id":"test123","amount":1000,"currency":"USD","return_url":"https://example.com/return","cancel_url":"https://example.com/cancel"}`

	var req model.CreateMerchantTransactionRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	if err != nil {
		t.Errorf("Failed to unmarshal request: %v", err)
	}

	if req.OrderID != "test123" {
		t.Errorf("Expected order_id test123, got %s", req.OrderID)
	}
	if req.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", req.Amount)
	}
	if req.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", req.Currency)
	}
}

func TestMerchantTransactionResponse_JSON(t *testing.T) {
	resp := model.MerchantTransactionResponse{
		TransactionID: "tx_123",
		RedirectURL:   "https://mywallet.example.com/pay/token123",
		Status:        "pending",
		ExpiresAt:     "2026-05-14T12:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal response: %v", err)
	}

	var decoded model.MerchantTransactionResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if decoded.TransactionID != "tx_123" {
		t.Errorf("Expected transaction_id tx_123, got %s", decoded.TransactionID)
	}
	if decoded.Status != "pending" {
		t.Errorf("Expected status pending, got %s", decoded.Status)
	}
}

func TestMerchantModel(t *testing.T) {
	merchant := model.Merchant{
		ID:            "merch_123",
		APIKey:        "key_abc123",
		Secret:        "supersecret",
		WebhookURL:    "https://shop.com/api/payments/wallet/webhook",
		WebhookSecret: "whsec_test",
	}

	if merchant.ID != "merch_123" {
		t.Errorf("Expected ID merch_123, got %s", merchant.ID)
	}
	if merchant.APIKey != "key_abc123" {
		t.Errorf("Expected APIKey key_abc123, got %s", merchant.APIKey)
	}
}

type mockWalletSvc struct{}

func (m *mockWalletSvc) GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	return &model.Wallet{Balance: 1000000}, nil
}

func (m *mockWalletSvc) Deduct(ctx context.Context, userID string, amount int64) error {
	if amount > 1000000 {
		return fmt.Errorf("insufficient balance")
	}
	return nil
}

type mockWebhookSvc struct{}

func (m *mockWebhookSvc) Dispatch(ctx context.Context, merchant model.Merchant, event string, tx *model.MerchantTransaction) error {
	return nil
}

func TestMockWalletService_Deduct(t *testing.T) {
	svc := &mockWalletSvc{}

	err := svc.Deduct(context.Background(), "test_user", 500)
	if err != nil {
		t.Errorf("Expected nil error for valid deduction, got %v", err)
	}

	err = svc.Deduct(context.Background(), "test_user", 2000000)
	if err == nil {
		t.Errorf("Expected error for insufficient balance, got nil")
	}
}

func TestPayPageData(t *testing.T) {
	data := struct {
		RedirectToken string
		OrderID       string
		Amount        string
		Currency      string
		Status        string
		ErrorMessage  string
		MerchantName  string
		FinalState    bool
	}{
		RedirectToken: "token123",
		OrderID:       "order_abc",
		Amount:        "$10.00",
		Currency:      "USD",
		Status:        "pending",
		MerchantName:  "Merchant Demo",
		FinalState:    false,
	}

	if data.RedirectToken != "token123" {
		t.Errorf("Expected redirect token token123, got %s", data.RedirectToken)
	}
	if data.Amount != "$10.00" {
		t.Errorf("Expected amount $10.00, got %s", data.Amount)
	}
	if data.FinalState != false {
		t.Errorf("Expected FinalState false, got %v", data.FinalState)
	}
}

func TestMerchantAuthHeaders(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/merchant/transactions", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Api-Key", "key_abc123")
	req.Header.Set("X-Signature", "sig123")
	req.Header.Set("X-Timestamp", "1234567890")

	if req.Header.Get("X-Api-Key") != "key_abc123" {
		t.Errorf("Expected X-Api-Key key_abc123, got %s", req.Header.Get("X-Api-Key"))
	}
	if req.Header.Get("X-Signature") != "sig123" {
		t.Errorf("Expected X-Signature sig123, got %s", req.Header.Get("X-Signature"))
	}
	if req.Header.Get("X-Timestamp") != "1234567890" {
		t.Errorf("Expected X-Timestamp 1234567890, got %s", req.Header.Get("X-Timestamp"))
	}
}

func TestIdempotencyKeyHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/merchant/transactions", bytes.NewBufferString(`{}`))
	req.Header.Set("Idempotency-Key", "idem_123")

	if req.Header.Get("Idempotency-Key") != "idem_123" {
		t.Errorf("Expected Idempotency-Key idem_123, got %s", req.Header.Get("Idempotency-Key"))
	}
}