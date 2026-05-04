package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken stores server-side refresh tokens for token rotation.
// Used is set to true when the token is consumed; expired or used tokens are rejected.
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(512);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"not null;default:false"`
	CreatedAt time.Time
}

// BeforeCreate hook — generates a UUID before inserting a new RefreshToken.
func (r *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// IsExpired returns true if the token has passed its expiry time.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
