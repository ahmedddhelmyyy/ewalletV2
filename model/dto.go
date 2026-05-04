package model

import (
	"time"

	"github.com/google/uuid"
)

// ─── Auth DTOs ────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ─── Auth Response DTOs ───────────────────────────────────────────────────────

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenPair struct {
	AccessToken             string    `json:"access_token"`
	RefreshToken            string    `json:"refresh_token"`
	AccessTokenExpiresAt    time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt   time.Time `json:"refresh_token_expires_at"`
}

type RegisterResponse struct {
	User   UserResponse `json:"user"`
	Wallet WalletResponse `json:"wallet"`
	Tokens TokenPair    `json:"tokens"`
}

type LoginResponse struct {
	User   UserResponse `json:"user"`
	Tokens TokenPair    `json:"tokens"`
}

type RefreshTokenResponse struct {
	Tokens TokenPair `json:"tokens"`
}

// ─── Wallet Response DTOs ─────────────────────────────────────────────────────

type WalletOwnerResponse struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

type WalletResponse struct {
	ID           uuid.UUID           `json:"id"`
	WalletNumber string              `json:"wallet_number"`
	Balance      int64               `json:"balance"`
	Currency     string              `json:"currency"`
	Owner        *WalletOwnerResponse `json:"owner,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// ─── Transaction Request DTOs ─────────────────────────────────────────────────

type SendRequest struct {
	RecipientWalletNumber string  `json:"recipient_wallet_number"`
	Amount                int64   `json:"amount"`
	Category              string  `json:"category"`
	Note                  *string `json:"note"`
}

type TopUpRequest struct {
	Amount int64   `json:"amount"`
	Note   *string `json:"note"`
}

type WithdrawRequest struct {
	Amount int64   `json:"amount"`
	Note   *string `json:"note"`
}

// ─── Transaction Filter / Pagination ─────────────────────────────────────────

type TransactionFilters struct {
	Type     string
	Category string
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

// ─── Transaction Response DTOs ────────────────────────────────────────────────

type TransactionResponse struct {
	ID                    uuid.UUID  `json:"id"`
	Type                  string     `json:"type"`
	Status                string     `json:"status"`
	Amount                int64      `json:"amount"`
	Currency              string     `json:"currency"`
	Category              string     `json:"category"`
	Note                  *string    `json:"note"`
	SenderWalletNumber    *string    `json:"sender_wallet_number"`
	RecipientWalletNumber *string    `json:"recipient_wallet_number"`
	CounterpartName       *string    `json:"counterpart_name"`
	CreatedAt             time.Time  `json:"created_at"`
}

type SendResponse struct {
	Transaction TransactionResponse `json:"transaction"`
	NewBalance  int64               `json:"new_balance"`
}

type TopUpResponse struct {
	Transaction TransactionResponse `json:"transaction"`
	NewBalance  int64               `json:"new_balance"`
}

type WithdrawResponse struct {
	Transaction TransactionResponse `json:"transaction"`
	NewBalance  int64               `json:"new_balance"`
}

// ─── Bill Request DTOs ────────────────────────────────────────────────────────

type CreateBillRequest struct {
	Name     string  `json:"name"`
	Amount   int64   `json:"amount"`
	DueDate  string  `json:"due_date"` // YYYY-MM-DD
	Category string  `json:"category"`
	Notes    *string `json:"notes"`
}

type BillFilters struct {
	Status   string
	Page     int
	PageSize int
}

// ─── Bill Response DTOs ───────────────────────────────────────────────────────

type BillResponse struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Name          string     `json:"name"`
	Amount        int64      `json:"amount"`
	Currency      string     `json:"currency"`
	DueDate       string     `json:"due_date"` // YYYY-MM-DD
	Category      string     `json:"category"`
	Status        string     `json:"status"`
	Notes         *string    `json:"notes"`
	PaidAt        *time.Time `json:"paid_at"`
	TransactionID *uuid.UUID `json:"transaction_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PayBillResponse struct {
	Bill        BillSummary         `json:"bill"`
	Transaction TransactionResponse `json:"transaction"`
	NewBalance  int64               `json:"new_balance"`
}

type BillSummary struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Amount        int64      `json:"amount"`
	Status        string     `json:"status"`
	PaidAt        *time.Time `json:"paid_at"`
	TransactionID *uuid.UUID `json:"transaction_id"`
}

// ─── Expense DTOs ─────────────────────────────────────────────────────────────

type ExpensePeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type CategoryBreakdown struct {
	Category   string  `json:"category"`
	Total      int64   `json:"total"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ExpenseSummaryResponse struct {
	Period     ExpensePeriod       `json:"period"`
	TotalSpent int64               `json:"total_spent"`
	Currency   string              `json:"currency"`
	Breakdown  []CategoryBreakdown `json:"breakdown"`
}

type FlowTotals struct {
	TotalIn  int64 `json:"total_in"`
	TotalOut int64 `json:"total_out"`
	Net      int64 `json:"net"`
}

type DailyFlowEntry struct {
	Date     string `json:"date"`
	MoneyIn  int64  `json:"money_in"`
	MoneyOut int64  `json:"money_out"`
}

type MoneyFlowResponse struct {
	Period   ExpensePeriod    `json:"period"`
	Currency string           `json:"currency"`
	Totals   FlowTotals       `json:"totals"`
	Daily    []DailyFlowEntry `json:"daily"`
}

// ─── Pagination ───────────────────────────────────────────────────────────────

// PaginationParams holds validated page and page_size values.
type PaginationParams struct {
	Page     int
	PageSize int
}

// Offset returns the SQL OFFSET value for the current page.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// TotalPages calculates the number of pages for a given total count.
func TotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}
