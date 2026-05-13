package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Transaction records every money movement in the system.
// Wallet numbers and counterpart names are denormalised for fast reads without joins.
type Transaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Wallet references — nullable to support top-ups (no sender) and withdrawals (no recipient)
	SenderWalletID    *uuid.UUID `gorm:"type:uuid;index"`
	RecipientWalletID *uuid.UUID `gorm:"type:uuid;index"`

	// Denormalised wallet numbers for display — avoids joins on history queries
	SenderWalletNumber    *string `gorm:"type:varchar(50)"`
	RecipientWalletNumber *string `gorm:"type:varchar(50)"`

	// Denormalised counterpart full name (the other party in the transaction)
	CounterpartName *string `gorm:"type:varchar(100)"`

	Type     string `gorm:"type:varchar(20);not null;index"`
	Status   string `gorm:"type:varchar(20);not null;default:'completed'"`
	Amount   int64  `gorm:"not null"`
	Currency string `gorm:"type:varchar(10);not null;default:'USD'"`
	Category string `gorm:"type:varchar(30);not null;index"`
	Note     *string `gorm:"type:varchar(255)"`

	CreatedAt time.Time `gorm:"index"`

	ExternalID *string `gorm:"type:varchar(100);uniqueIndex"`
}

// BeforeCreate hook — generates a UUID before inserting a new Transaction.
func (t *Transaction) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
