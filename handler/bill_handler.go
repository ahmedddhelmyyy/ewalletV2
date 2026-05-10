package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/service"
)

// BillHandler holds HTTP handlers for bill endpoints.
type BillHandler struct {
	billSvc service.BillService
}

// NewBillHandler creates a new BillHandler.
func NewBillHandler(billSvc service.BillService) *BillHandler {
	return &BillHandler{billSvc: billSvc}
}

// Create godoc
//
//	@Summary		Create bill
//	@Description	Creates a new pending bill for the authenticated user.
//	@Tags			Bills
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		model.CreateBillRequest	true	"Bill details"
//	@Success		201		{object}	SuccessEnvelope{data=model.BillResponse}
//	@Failure		400		{object}	ErrorEnvelope
//	@Failure		401		{object}	ErrorEnvelope
//	@Router			/bills [post]
func (h *BillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBillRequest
	if err := decodeJSON(r, &req); err != nil {
		validationError(w, map[string]string{"body": "Invalid JSON payload."})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	errs := map[string]string{}
	if len(req.Name) < 2 || len(req.Name) > 200 {
		errs["name"] = "Must be between 2 and 200 characters."
	}
	if req.Amount < 1 {
		errs["amount"] = "Amount must be at least 1 cent."
	}
	if req.DueDate == "" {
		errs["due_date"] = "Due date is required (YYYY-MM-DD)."
	} else if _, err := time.Parse("2006-01-02", req.DueDate); err != nil {
		errs["due_date"] = "Must be a valid date in YYYY-MM-DD format."
	}
	if req.Category != "" && !model.ValidCategories[req.Category] {
		errs["category"] = "Invalid category value."
	}
	if req.Notes != nil && len(*req.Notes) > 500 {
		errs["notes"] = "Notes must not exceed 500 characters."
	}
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.billSvc.Create(userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, resp)
}

// List godoc
//
//	@Summary		List bills
//	@Description	Returns paginated bills for the authenticated user. Results are sorted by due_date ascending (soonest first).
//	@Tags			Bills
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"Filter by status"	Enums(pending, paid)
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			page_size	query		int		false	"Items per page (default: 20, max: 100)"
//	@Success		200			{object}	PaginatedEnvelope{data=[]model.BillResponse}
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		401			{object}	ErrorEnvelope
//	@Router			/bills [get]
func (h *BillHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	statusFilter := r.URL.Query().Get("status")

	if statusFilter != "" && statusFilter != model.BillStatusPending && statusFilter != model.BillStatusPaid {
		validationError(w, map[string]string{"status": "Must be 'pending' or 'paid'."})
		return
	}

	userID := userIDFromCtx(r)
	bills, total, err := h.billSvc.List(userID, model.BillFilters{
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondPaginated(w, page, pageSize, total, bills)
}

// GetByID godoc
//
//	@Summary		Get single bill
//	@Description	Returns details for one bill. The bill must belong to the authenticated user.
//	@Tags			Bills
//	@Produce		json
//	@Security		BearerAuth
//	@Param			bill_id	path		string	true	"Bill UUID"
//	@Success		200		{object}	SuccessEnvelope{data=model.BillResponse}
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		403		{object}	ErrorEnvelope
//	@Failure		404		{object}	ErrorEnvelope
//	@Router			/bills/{bill_id} [get]
func (h *BillHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	billID, err := uuid.Parse(chi.URLParam(r, "bill_id"))
	if err != nil {
		validationError(w, map[string]string{"bill_id": "Must be a valid UUID."})
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.billSvc.GetByID(userID, billID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}

// Pay godoc
//
//	@Summary		Pay bill
//	@Description	Marks a pending bill as paid and atomically deducts the bill amount from the wallet. The wallet deduction and bill status update happen in a single DB transaction.
//	@Tags			Bills
//	@Produce		json
//	@Security		BearerAuth
//	@Param			bill_id	path		string	true	"Bill UUID"
//	@Success		200		{object}	SuccessEnvelope{data=model.PayBillResponse}
//	@Failure		400		{object}	ErrorEnvelope	"Already paid / insufficient funds"
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		403		{object}	ErrorEnvelope
//	@Failure		404		{object}	ErrorEnvelope
//	@Router			/bills/{bill_id}/pay [post]
func (h *BillHandler) Pay(w http.ResponseWriter, r *http.Request) {
	billID, err := uuid.Parse(chi.URLParam(r, "bill_id"))
	if err != nil {
		validationError(w, map[string]string{"bill_id": "Must be a valid UUID."})
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.billSvc.Pay(userID, billID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}

// Delete godoc
//
//	@Summary		Delete bill
//	@Description	Permanently deletes a pending bill. Paid bills are locked and cannot be deleted (financial audit trail).
//	@Tags			Bills
//	@Produce		json
//	@Security		BearerAuth
//	@Param			bill_id	path		string	true	"Bill UUID"
//	@Success		200		{object}	SuccessEnvelope
//	@Failure		400		{object}	ErrorEnvelope	"Cannot delete paid bill"
//	@Failure		401		{object}	ErrorEnvelope
//	@Failure		403		{object}	ErrorEnvelope
//	@Failure		404		{object}	ErrorEnvelope
//	@Router			/bills/{bill_id} [delete]
func (h *BillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	billID, err := uuid.Parse(chi.URLParam(r, "bill_id"))
	if err != nil {
		validationError(w, map[string]string{"bill_id": "Must be a valid UUID."})
		return
	}

	userID := userIDFromCtx(r)
	if err := h.billSvc.Delete(userID, billID); err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "Bill deleted successfully."})
}