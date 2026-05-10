package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/service"
)

// TransactionHandler holds HTTP handlers for transaction endpoints.
type TransactionHandler struct {
	txSvc service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(txSvc service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txSvc: txSvc}
}

// Send godoc
//
//	@Summary		Send money
//	@Description	Transfers money from the caller's wallet to another wallet by wallet number. Both the debit and credit are atomic — if either fails the whole operation is rolled back.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		model.SendRequest	true	"Transfer details"
//	@Success		201		{object}	SuccessEnvelope{data=model.SendResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Validation error / insufficient funds / self-transfer"
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		404		{object}	ErrorEnvelope	"Recipient wallet not found"
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/transactions/send [post]
func (h *TransactionHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req model.SendRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	errs := map[string]string{}
	if req.RecipientWalletNumber == "" {
		errs["recipient_wallet_number"] = "Recipient wallet number is required."
	}
	if req.Amount < 1 {
		errs["amount"] = "Amount must be at least 1 cent."
	}
	if req.Category != "" && !model.ValidCategories[req.Category] {
		errs["category"] = "Invalid category value."
	}
	if req.Note != nil && len(*req.Note) > 255 {
		errs["note"] = "Note must not exceed 255 characters."
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.txSvc.Send(userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, resp)
}

// TopUp godoc
//
//	@Summary		Top-up / Deposit
//	@Description	Adds money to the caller's own wallet. Simulates an external bank deposit (no real payment gateway in v1). Min: $1.00 (100 cents). Max: $100,000.00 (10,000,000 cents).
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		model.TopUpRequest	true	"Top-up details"
//	@Success		201		{object}	SuccessEnvelope{data=model.TopUpResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Amount out of range"
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/transactions/top-up [post]
func (h *TransactionHandler) TopUp(w http.ResponseWriter, r *http.Request) {
	var req model.TopUpRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	errs := map[string]string{}
	if req.Amount < model.MinTopUpAmount {
		errs["amount"] = "Amount must be at least 100 cents ($1.00)."
	}
	if req.Amount > model.MaxTopUpAmount {
		errs["amount"] = "Amount must not exceed 10,000,000 cents ($100,000.00)."
	}
	if req.Note != nil && len(*req.Note) > 255 {
		errs["note"] = "Note must not exceed 255 characters."
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.txSvc.TopUp(userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, resp)
}

// Withdraw godoc
//
//	@Summary		Withdraw
//	@Description	Deducts money from the caller's wallet. Simulates a bank withdrawal. Min: $1.00 (100 cents).
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		model.WithdrawRequest	true	"Withdrawal details"
//	@Success		201		{object}	SuccessEnvelope{data=model.WithdrawResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Insufficient funds"
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/transactions/withdraw [post]
func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var req model.WithdrawRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	errs := map[string]string{}
	if req.Amount < model.MinWithdrawAmount {
		errs["amount"] = "Amount must be at least 100 cents ($1.00)."
	}
	if req.Note != nil && len(*req.Note) > 255 {
		errs["note"] = "Note must not exceed 255 characters."
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.txSvc.Withdraw(userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, resp)
}

// GetHistory godoc
//
//	@Summary		Transaction history
//	@Description	Returns paginated transaction history for the caller's wallet. Sorted by date descending (newest first). Supports filtering by type, category, and date range.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			page_size	query		int		false	"Items per page (default: 20, max: 100)"
//	@Param			type		query		string	false	"Filter by type"		Enums(send, receive, top_up, withdrawal, bill_payment)
//	@Param			category	query		string	false	"Filter by category"	Enums(food, transport, bills, transfer, top_up, withdrawal, other)
//	@Param			from		query		string	false	"Start date — ISO 8601"	example(2026-05-01T00:00:00Z)
//	@Param			to			query		string	false	"End date — ISO 8601"	example(2026-05-31T23:59:59Z)
//	@Success		200			{object}	PaginatedEnvelope{data=[]model.TransactionResponse}
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		401			{object}	ErrorEnvelope
//	@Router			/transactions [get]
func (h *TransactionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	q := r.URL.Query()
	filters := model.TransactionFilters{Page: page, PageSize: pageSize}
	errs := map[string]string{}

	if txType := q.Get("type"); txType != "" {
		if !model.ValidTxTypes[txType] {
			errs["type"] = "Invalid transaction type value."
		} else {
			filters.Type = txType
		}
	}
	if cat := q.Get("category"); cat != "" {
		if !model.ValidCategories[cat] {
			errs["category"] = "Invalid category value."
		} else {
			filters.Category = cat
		}
	}
	if fromStr := q.Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			errs["from"] = "Must be ISO 8601, e.g. 2026-05-01T00:00:00Z."
		} else {
			filters.From = &t
		}
	}
	if toStr := q.Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			errs["to"] = "Must be ISO 8601, e.g. 2026-05-31T23:59:59Z."
		} else {
			filters.To = &t
		}
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	txs, total, err := h.txSvc.GetHistory(userID, filters)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondPaginated(w, page, pageSize, total, txs)
}

// GetByID godoc
//
//	@Summary		Get single transaction
//	@Description	Returns details for one transaction. The caller must be either the sender or the recipient — returns 403 otherwise.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			transaction_id	path		string	true	"Transaction UUID"
//	@Success		200				{object}	SuccessEnvelope{data=model.TransactionResponse}
//	@Failure		401				{object}	ErrorEnvelope
//	@Failure		403				{object}	ErrorEnvelope
//	@Failure		404				{object}	ErrorEnvelope
//	@Router			/transactions/{transaction_id} [get]
func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	txID, err := uuid.Parse(chi.URLParam(r, "transaction_id"))
	if err != nil {
		validationError(w, map[string]string{"transaction_id": "Must be a valid UUID."})
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.txSvc.GetByID(userID, txID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}