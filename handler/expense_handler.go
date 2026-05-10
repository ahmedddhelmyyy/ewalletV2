package handler

import (
	"net/http"
	"time"

	"github.com/ewallet/model"
	"github.com/ewallet/service"
)

// ExpenseHandler holds HTTP handlers for expense analytics endpoints.
type ExpenseHandler struct {
	expenseSvc service.ExpenseService
}

// NewExpenseHandler creates a new ExpenseHandler.
func NewExpenseHandler(expenseSvc service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseSvc: expenseSvc}
}

// GetSummary handles GET /expenses/summary
func (h *ExpenseHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	from, to, errs := parseDateRange(r)
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.expenseSvc.GetSummary(userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, resp)
}

// GetFlow handles GET /expenses/flow
func (h *ExpenseHandler) GetFlow(w http.ResponseWriter, r *http.Request) {
	from, to, errs := parseDateRange(r)
	if len(errs) > 0 {
		validationError(w, errs)
		return
	}

	// Enforce 365-day max range
	if to.Sub(from).Hours()/24 > model.MaxFlowRangeDays {
		validationError(w, map[string]string{"range": "Date range must not exceed 365 days."})
		return
	}

	userID := userIDFromCtx(r)
	resp, err := h.expenseSvc.GetFlow(userID, from, to)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, resp)
}

// parseDateRange extracts and validates the ?from=&to= query parameters.
// Both are required and from must not be after to.
func parseDateRange(r *http.Request) (from, to time.Time, errs map[string]string) {
	errs = map[string]string{}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" {
		errs["from"] = "from is required (ISO 8601, e.g. 2026-05-01T00:00:00Z)."
	} else {
		var err error
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			errs["from"] = "Must be ISO 8601 format, e.g. 2026-05-01T00:00:00Z."
		}
	}

	if toStr == "" {
		errs["to"] = "to is required (ISO 8601, e.g. 2026-05-31T23:59:59Z)."
	} else {
		var err error
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			errs["to"] = "Must be ISO 8601 format, e.g. 2026-05-31T23:59:59Z."
		}
	}

	if len(errs) == 0 && from.After(to) {
		errs["range"] = "from must not be after to."
	}

	return from, to, errs
}
