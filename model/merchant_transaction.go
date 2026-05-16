package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MerchantTransaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	MerchantID      string    `gorm:"type:varchar(50);not null;index"`
	MerchantUserID  string    `gorm:"type:varchar(50)"`
	OrderID         string    `gorm:"type:varchar(100);not null"`
	IdempotencyKey  string    `gorm:"type:varchar(100);index"`
	Amount          int64     `gorm:"not null"`
	Currency        string    `gorm:"type:varchar(3);not null;default:'USD'"`
	Status          string    `gorm:"type:varchar(20);not null;default:'pending'"` // pending, success, failed, expired, refunded
	ReturnURL       string    `gorm:"type:text;not null"`
	CancelURL       string    `gorm:"type:text"`
	RedirectToken   string    `gorm:"type:varchar(100);uniqueIndex"`
	UserID          string    `gorm:"type:varchar(50)"`
	WebhookURL      string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"index"`
	UpdatedAt       time.Time
	ExpiresAt       time.Time `gorm:"index"`
}

func (m *MerchantTransaction) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.RedirectToken == "" {
		m.RedirectToken = uuid.New().String()
	}
	if m.ExpiresAt.IsZero() {
		m.ExpiresAt = time.Now().Add(15 * time.Minute)
	}
	return nil
}

type CreateMerchantTransactionRequest struct {
	OrderID    string `json:"order_id" required:"true"`
	Amount     int64  `json:"amount" required:"true"`
	Currency   string `json:"currency" required:"true"`
	ReturnURL  string `json:"return_url" required:"true"`
	CancelURL  string `json:"cancel_url"`
	WebhookURL string `json:"webhook_url"`
}

type MerchantTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	RedirectURL   string `json:"redirect_url"`
	Status        string `json:"status"`
	ExpiresAt     string `json:"expires_at"`
}