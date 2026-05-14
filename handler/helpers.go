///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/handler/helpers.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/ewallet/model"
)

type contextKey string

const ContextKeyUserID string = "userID"

func userIDFromCtx(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(ContextKeyUserID).(uuid.UUID)
	return id
}

// ─── Response envelopes (exported so swag can generate schemas) ───────────────

// SuccessEnvelope is the standard wrapper for all successful single-item responses.
type SuccessEnvelope struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data"`
}

// PaginatedEnvelope is the standard wrapper for all paginated list responses.
type PaginatedEnvelope struct {
	Success    bool        `json:"success"     example:"true"`
	Page       int         `json:"page"        example:"1"`
	PageSize   int         `json:"page_size"   example:"20"`
	Total      int64       `json:"total"       example:"87"`
	TotalPages int         `json:"total_pages" example:"5"`
	Data       interface{} `json:"data"`
}

// ErrorDetail is the inner error object inside ErrorEnvelope.
type ErrorDetail struct {
	Code    string      `json:"code"              example:"VALIDATION_ERROR"`
	Message string      `json:"message"           example:"Request validation failed."`
	Details interface{} `json:"details,omitempty"`
}

// ErrorEnvelope is the standard wrapper for all error responses.
type ErrorEnvelope struct {
	Success bool        `json:"success" example:"false"`
	Error   ErrorDetail `json:"error"`
}

// ─── Writers ─────────────────────────────────────────────────────────────────

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondSuccess(w http.ResponseWriter, status int, data interface{}) {
	respondJSON(w, status, SuccessEnvelope{Success: true, Data: data})
}

func respondPaginated(w http.ResponseWriter, page, pageSize int, total int64, data interface{}) {
	respondJSON(w, http.StatusOK, PaginatedEnvelope{
		Success:    true,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: model.TotalPages(total, pageSize),
		Data:       data,
	})
}

func respondError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	respondJSON(w, status, ErrorEnvelope{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// ─── Error mapper ─────────────────────────────────────────────────────────────

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrEmailAlreadyExists):
		respondError(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "An account with this email already exists.", nil)
	case errors.Is(err, model.ErrInvalidCredentials):
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password.", nil)
	case errors.Is(err, model.ErrInvalidRefreshToken):
		respondError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "The refresh token is invalid, expired, or has already been used.", nil)
	case errors.Is(err, model.ErrTokenNotFound):
		respondError(w, http.StatusNotFound, "TOKEN_NOT_FOUND", "Refresh token not found.", nil)
	case errors.Is(err, model.ErrUserNotFound):
		respondError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found.", nil)
	case errors.Is(err, model.ErrWalletNotFound):
		respondError(w, http.StatusNotFound, "WALLET_NOT_FOUND", "Wallet not found.", nil)
	case errors.Is(err, model.ErrTransactionNotFound):
		respondError(w, http.StatusNotFound, "TRANSACTION_NOT_FOUND", "Transaction not found.", nil)
	case errors.Is(err, model.ErrBillNotFound):
		respondError(w, http.StatusNotFound, "BILL_NOT_FOUND", "Bill not found.", nil)
	case errors.Is(err, model.ErrInsufficientFunds):
		respondError(w, http.StatusBadRequest, "INSUFFICIENT_FUNDS", "Insufficient funds to complete this operation.", nil)
	case errors.Is(err, model.ErrSelfTransfer):
		respondError(w, http.StatusBadRequest, "SELF_TRANSFER", "Cannot transfer money to your own wallet.", nil)
	case errors.Is(err, model.ErrBillAlreadyPaid):
		respondError(w, http.StatusBadRequest, "BILL_ALREADY_PAID", "This bill has already been paid.", nil)
	case errors.Is(err, model.ErrCannotDeletePaid):
		respondError(w, http.StatusBadRequest, "CANNOT_DELETE_PAID_BILL", "Paid bills cannot be deleted.", nil)
	case errors.Is(err, model.ErrForbidden):
		respondError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to access this resource.", nil)
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", nil)
	}
}

// ─── Request helpers ──────────────────────────────────────────────────────────

func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page, pageSize = 1, 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}
	return page, pageSize
}

func validationError(w http.ResponseWriter, details map[string]string) {
	respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed.", details)
}