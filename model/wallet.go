package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Wallet represents the financial wallet owned by a single user.
// Balance is stored in cents (integer) — never floats.
type Wallet struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	WalletNumber string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Balance      int64     `gorm:"not null;default:0"`
	Currency     string    `gorm:"type:varchar(10);not null;default:'USD'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Belongs-to association
	User *User `gorm:"foreignKey:UserID"`
}

// BeforeCreate hook — generates a UUID before inserting a new Wallet.
func (w *Wallet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
