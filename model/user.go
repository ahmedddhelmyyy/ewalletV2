package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents an application user stored in the `users` table.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName  string    `gorm:"type:varchar(100);not null"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Has-one association — loaded explicitly where needed
	Wallet *Wallet `gorm:"foreignKey:UserID"`
}

// BeforeCreate hook — generates a UUID before inserting a new User.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
