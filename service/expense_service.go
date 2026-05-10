package service

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
)

type expenseService struct {
	walletRepo repository.WalletRepository
	txRepo     repository.TransactionRepository
}

// NewExpenseService creates a new ExpenseService.
func NewExpenseService(
	walletRepo repository.WalletRepository,
	txRepo repository.TransactionRepository,
) ExpenseService {
	return &expenseService{
		walletRepo: walletRepo,
		txRepo:     txRepo,
	}
}

// GetSummary returns total spend per category within the given date range.
func (s *expenseService) GetSummary(userID uuid.UUID, from, to time.Time) (*model.ExpenseSummaryResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.txRepo.GetCategorySummary(wallet.ID, from, to)
	if err != nil {
		return nil, err
	}

	var totalSpent int64
	for _, r := range rows {
		totalSpent += r.Total
	}

	breakdown := make([]model.CategoryBreakdown, 0, len(rows))
	for _, r := range rows {
		var pct float64
		if totalSpent > 0 {
			pct = math.Round(float64(r.Total)/float64(totalSpent)*10000) / 100
		}
		breakdown = append(breakdown, model.CategoryBreakdown{
			Category:   r.Category,
			Total:      r.Total,
			Count:      r.TxCount,
			Percentage: pct,
		})
	}

	return &model.ExpenseSummaryResponse{
		Period:     model.ExpensePeriod{From: from, To: to},
		TotalSpent: totalSpent,
		Currency:   model.DefaultCurrency,
		Breakdown:  breakdown,
	}, nil
}

// GetFlow returns daily money-in vs money-out for every calendar day in the range.
// Days with no activity are included with zeros to simplify chart rendering.
func (s *expenseService) GetFlow(userID uuid.UUID, from, to time.Time) (*model.MoneyFlowResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	inRows, err := s.txRepo.GetDailyInFlow(wallet.ID, from, to)
	if err != nil {
		return nil, err
	}
	outRows, err := s.txRepo.GetDailyOutFlow(wallet.ID, from, to)
	if err != nil {
		return nil, err
	}

	// Build lookup maps by date string
	inMap := make(map[string]int64, len(inRows))
	for _, r := range inRows {
		inMap[r.Date] = r.Amount
	}
	outMap := make(map[string]int64, len(outRows))
	for _, r := range outRows {
		outMap[r.Date] = r.Amount
	}

	// Iterate every calendar day in [from, to] and produce an entry
	var (
		daily    []model.DailyFlowEntry
		totalIn  int64
		totalOut int64
	)

	current := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		moneyIn := inMap[dateStr]
		moneyOut := outMap[dateStr]
		daily = append(daily, model.DailyFlowEntry{
			Date:     dateStr,
			MoneyIn:  moneyIn,
			MoneyOut: moneyOut,
		})
		totalIn += moneyIn
		totalOut += moneyOut
		current = current.AddDate(0, 0, 1)
	}

	return &model.MoneyFlowResponse{
		Period:   model.ExpensePeriod{From: from, To: to},
		Currency: model.DefaultCurrency,
		Totals: model.FlowTotals{
			TotalIn:  totalIn,
			TotalOut: totalOut,
			Net:      totalIn - totalOut,
		},
		Daily: daily,
	}, nil
}
