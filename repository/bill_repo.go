package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"gorm.io/gorm"
)

type billRepository struct {
	db *gorm.DB
}

// NewBillRepository creates a new BillRepository.
func NewBillRepository(db *gorm.DB) BillRepository {
	return &billRepository{db: db}
}

func (r *billRepository) WithTx(tx *gorm.DB) BillRepository {
	return &billRepository{db: tx}
}

func (r *billRepository) Create(bill *model.Bill) error {
	return r.db.Create(bill).Error
}

func (r *billRepository) FindByID(id uuid.UUID) (*model.Bill, error) {
	var bill model.Bill
	err := r.db.First(&bill, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrBillNotFound
		}
		return nil, err
	}
	return &bill, nil
}

// FindByUserID returns paginated bills for a user with optional status filter.
// Results are sorted by due_date ascending (soonest due first).
func (r *billRepository) FindByUserID(userID uuid.UUID, filters model.BillFilters) ([]model.Bill, int64, error) {
	query := r.db.Model(&model.Bill{}).Where("user_id = ?", userID)

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var bills []model.Bill
	offset := (filters.Page - 1) * filters.PageSize
	err := query.
		Order("due_date ASC").
		Offset(offset).
		Limit(filters.PageSize).
		Find(&bills).Error

	return bills, total, err
}

// Save persists all changes to an existing bill record (full update).
func (r *billRepository) Save(bill *model.Bill) error {
	return r.db.Save(bill).Error
}

// Delete hard-deletes a bill by ID.
func (r *billRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Bill{}, "id = ?", id).Error
}
