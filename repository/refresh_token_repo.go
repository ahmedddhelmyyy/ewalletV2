package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *refreshTokenRepository) FindByToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token = ?", token).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenNotFound
		}
		return nil, err
	}
	return &rt, nil
}

// MarkUsed sets the Used flag to true so the token cannot be reused (token rotation).
func (r *refreshTokenRepository) MarkUsed(id uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("used", true).Error
}

// DeleteByUserID removes all refresh tokens for a user (used on logout).
func (r *refreshTokenRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.RefreshToken{}).Error
}
