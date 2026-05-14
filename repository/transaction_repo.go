package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ewallet/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new TransactionRepository.
func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) WithTx(tx *gorm.DB) TransactionRepository {
	return &transactionRepository{db: tx}
}

func (r *transactionRepository) Create(tx *model.Transaction) error {
	return r.db.Create(tx).Error
}

func (r *transactionRepository) FindByID(id uuid.UUID) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.First(&tx, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTransactionNotFound
		}
		return nil, err
	}
	return &tx, nil
}

// FindByWallet retrieves all transactions where the given wallet is sender or recipient,
// applying optional filters for type, category, and date range. Returns paginated results.
func (r *transactionRepository) FindByWallet(walletID uuid.UUID, filters model.TransactionFilters) ([]model.Transaction, int64, error) {
	query := r.db.Model(&model.Transaction{}).
		Where("sender_wallet_id = ? OR recipient_wallet_id = ?", walletID, walletID)

	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.From != nil {
		query = query.Where("created_at >= ?", filters.From)
	}
	if filters.To != nil {
		query = query.Where("created_at <= ?", filters.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var transactions []model.Transaction
	offset := (filters.Page - 1) * filters.PageSize
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(filters.PageSize).
		Find(&transactions).Error

	return transactions, total, err
}

// GetCategorySummary aggregates total spend and transaction count per category
// for debit transactions (send, withdrawal, bill_payment) from the given wallet.
func (r *transactionRepository) GetCategorySummary(walletID uuid.UUID, from, to time.Time) ([]CategorySummaryRow, error) {
	var rows []CategorySummaryRow
	err := r.db.Raw(`
		SELECT
			category,
			SUM(amount)  AS total,
			COUNT(*)     AS tx_count
		FROM transactions
		WHERE sender_wallet_id = ?
		  AND type IN ('send', 'withdrawal', 'bill_payment')
		  AND created_at >= ?
		  AND created_at <= ?
		GROUP BY category
		ORDER BY total DESC
	`, walletID, from, to).Scan(&rows).Error
	return rows, err
}

// GetDailyInFlow aggregates daily credit amounts (receive, top_up) for the given wallet.
func (r *transactionRepository) GetDailyInFlow(walletID uuid.UUID, from, to time.Time) ([]DailyFlowRow, error) {
	var rows []DailyFlowRow
	err := r.db.Raw(`
		SELECT
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS date,
			SUM(amount) AS amount
		FROM transactions
		WHERE recipient_wallet_id = ?
		  AND type IN ('receive', 'top_up')
		  AND created_at >= ?
		  AND created_at <= ?
		GROUP BY TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')
	`, walletID, from, to).Scan(&rows).Error
	return rows, err
}

// GetDailyOutFlow aggregates daily debit amounts (send, withdrawal, bill_payment) for the given wallet.
func (r *transactionRepository) GetDailyOutFlow(walletID uuid.UUID, from, to time.Time) ([]DailyFlowRow, error) {
	var rows []DailyFlowRow
	err := r.db.Raw(`
		SELECT
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS date,
			SUM(amount) AS amount
		FROM transactions
		WHERE sender_wallet_id = ?
		  AND type IN ('send', 'withdrawal', 'bill_payment')
		  AND created_at >= ?
		  AND created_at <= ?
		GROUP BY TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')
	`, walletID, from, to).Scan(&rows).Error
	return rows, err
}

func (r *transactionRepository) ExistsByExternalID(ctx context.Context, externalID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Where("external_id = ?", externalID).
		Count(&count).Error
	return count > 0, err
}

func (r *transactionRepository) CreateWithBalanceCredit(ctx context.Context, tx *model.Transaction, walletID uuid.UUID, amount int64) error {
	return r.db.WithContext(ctx).Transaction(func(txDB *gorm.DB) error {
		if err := txDB.Create(tx).Error; err != nil {
			return err
		}
		return txDB.Exec(`
			UPDATE wallets
			SET balance = balance + ?
			WHERE id = ?
		`, amount, walletID).Error
	})
}
