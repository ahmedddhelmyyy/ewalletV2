package service

import (
	"fmt"
	"time"

	"github.com/ewallet/model"
	"github.com/ewallet/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type billService struct {
	db         *gorm.DB
	billRepo   repository.BillRepository
	walletRepo repository.WalletRepository
	txRepo     repository.TransactionRepository
}

// NewBillService creates a new BillService.
func NewBillService(
	db *gorm.DB,
	billRepo repository.BillRepository,
	walletRepo repository.WalletRepository,
	txRepo repository.TransactionRepository,
) BillService {
	return &billService{
		db:         db,
		billRepo:   billRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
	}
}

// Create persists a new bill for the given user.
func (s *billService) Create(userID uuid.UUID, req model.CreateBillRequest) (*model.BillResponse, error) {
	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid due_date format, expected YYYY-MM-DD: %w", err)
	}

	category := req.Category
	if category == "" {
		category = model.CategoryBills
	}

	bill := model.Bill{
		UserID:   userID,
		Name:     req.Name,
		Amount:   req.Amount,
		Currency: model.DefaultCurrency,
		DueDate:  dueDate,
		Category: category,
		Status:   model.BillStatusPending,
		Notes:    req.Notes,
	}

	if err := s.billRepo.Create(&bill); err != nil {
		return nil, err
	}

	resp := mapBillToResponse(&bill)
	return &resp, nil
}

// List returns paginated bills for the user with optional status filter.
func (s *billService) List(userID uuid.UUID, filters model.BillFilters) ([]model.BillResponse, int64, error) {
	bills, total, err := s.billRepo.FindByUserID(userID, filters)
	if err != nil {
		return nil, 0, err
	}

	result := make([]model.BillResponse, len(bills))
	for i, b := range bills {
		result[i] = mapBillToResponse(&b)
	}
	return result, total, nil
}

// GetByID returns a single bill, enforcing ownership.
func (s *billService) GetByID(userID uuid.UUID, billID uuid.UUID) (*model.BillResponse, error) {
	bill, err := s.billRepo.FindByID(billID)
	if err != nil {
		return nil, err
	}
	if bill.UserID != userID {
		return nil, model.ErrForbidden
	}
	resp := mapBillToResponse(bill)
	return &resp, nil
}

// Pay marks a pending bill as paid and atomically deducts its amount from the wallet.
func (s *billService) Pay(userID uuid.UUID, billID uuid.UUID) (*model.PayBillResponse, error) {
	bill, err := s.billRepo.FindByID(billID)
	if err != nil {
		return nil, err
	}
	if bill.UserID != userID {
		return nil, model.ErrForbidden
	}
	if bill.Status == model.BillStatusPaid {
		return nil, model.ErrBillAlreadyPaid
	}

	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if wallet.Balance < bill.Amount {
		return nil, model.ErrInsufficientFunds
	}

	newBalance := wallet.Balance - bill.Amount
	now := time.Now().UTC()
	note := fmt.Sprintf("Payment for: %s", bill.Name)

	var newTx model.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		wRepo := s.walletRepo.WithTx(tx)
		tRepo := s.txRepo.WithTx(tx)
		bRepo := s.billRepo.WithTx(tx)

		if err := wRepo.UpdateBalance(wallet.ID, newBalance); err != nil {
			return fmt.Errorf("debit wallet: %w", err)
		}

		newTx = model.Transaction{
			SenderWalletID:     &wallet.ID,
			SenderWalletNumber: &wallet.WalletNumber,
			Type:               model.TxTypeBillPayment,
			Status:             model.TxStatusCompleted,
			Amount:             bill.Amount,
			Currency:           model.DefaultCurrency,
			Category:           model.CategoryBills,
			Note:               &note,
		}
		if err := tRepo.Create(&newTx); err != nil {
			return fmt.Errorf("create bill payment tx: %w", err)
		}

		// Update bill
		bill.Status = model.BillStatusPaid
		bill.PaidAt = &now
		bill.TransactionID = &newTx.ID
		return bRepo.Save(bill)
	})
	if err != nil {
		return nil, err
	}

	txID := newTx.ID
	return &model.PayBillResponse{
		Bill: model.BillSummary{
			ID:            bill.ID,
			Name:          bill.Name,
			Amount:        bill.Amount,
			Status:        bill.Status,
			PaidAt:        bill.PaidAt,
			TransactionID: &txID,
		},
		Transaction: mapTransactionToResponse(&newTx),
		NewBalance:  newBalance,
	}, nil
}

// Delete hard-deletes a pending bill. Paid bills cannot be deleted.
func (s *billService) Delete(userID uuid.UUID, billID uuid.UUID) error {
	bill, err := s.billRepo.FindByID(billID)
	if err != nil {
		return err
	}
	if bill.UserID != userID {
		return model.ErrForbidden
	}
	if bill.Status == model.BillStatusPaid {
		return model.ErrCannotDeletePaid
	}
	return s.billRepo.Delete(billID)
}

// mapBillToResponse converts a Bill model to the API response DTO.
func mapBillToResponse(b *model.Bill) model.BillResponse {
	return model.BillResponse{
		ID:            b.ID,
		UserID:        b.UserID,
		Name:          b.Name,
		Amount:        b.Amount,
		Currency:      b.Currency,
		DueDate:       b.DueDate.Format("2006-01-02"),
		Category:      b.Category,
		Status:        b.Status,
		Notes:         b.Notes,
		PaidAt:        b.PaidAt,
		TransactionID: b.TransactionID,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}
}
