package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Bill represents a payable bill tracked by a user.
type Bill struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Name     string    `gorm:"type:varchar(200);not null"`
	Amount   int64     `gorm:"not null"`
	Currency string    `gorm:"type:varchar(10);not null;default:'USD'"`
	DueDate  time.Time `gorm:"not null"`
	Category string    `gorm:"type:varchar(30);not null;default:'bills'"`
	Status   string    `gorm:"type:varchar(20);not null;default:'pending';index"`
	Notes    *string   `gorm:"type:varchar(500)"`

	// Populated once the bill is paid
	PaidAt        *time.Time
	TransactionID *uuid.UUID `gorm:"type:uuid"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeCreate hook — generates a UUID before inserting a new Bill.
func (b *Bill) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
