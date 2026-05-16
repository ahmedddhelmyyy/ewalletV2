package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ewallet/service"
)

type PayHandler struct {
	svc      service.MerchantServiceInterface
	renderer *Renderer
}

func NewPayHandler(svc service.MerchantServiceInterface, renderer *Renderer) *PayHandler {
	return &PayHandler{svc: svc, renderer: renderer}
}

func (h *PayHandler) Show(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "redirect_token")
	ctx := r.Context()

	tx, err := h.svc.GetTransactionByToken(ctx, token)
	if err != nil || tx == nil {
		h.renderer.Execute(w, "error.html", PayPageData{ErrorMessage: "Transaction not found"})
		return
	}

	if time.Now().After(tx.ExpiresAt) && tx.Status == "pending" {
		h.svc.ExpireTransaction(ctx, tx.ID.String())
		h.renderer.Execute(w, "expired.html", PayPageData{
			RedirectToken: token,
			MerchantName:  "Merchant Demo",
		})
		return
	}

	if tx.Status != "pending" {
		h.renderer.Execute(w, "pay.html", PayPageData{
			RedirectToken: token,
			OrderID:       tx.OrderID,
			Amount:        formatAmount(tx.Amount),
			Currency:      tx.Currency,
			ExpiresAt:     tx.ExpiresAt,
			Status:        tx.Status,
			MerchantName:  "Merchant Demo",
			FinalState:    true,
		})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test_user"
	}

	h.renderer.Execute(w, "pay.html", PayPageData{
		RedirectToken: token,
		OrderID:       tx.OrderID,
		Amount:        formatAmount(tx.Amount),
		Currency:      tx.Currency,
		ExpiresAt:     tx.ExpiresAt,
		Status:        tx.Status,
		MerchantName:  "Merchant Demo",
		FinalState:    false,
		UserID:        userID,
	})
}

func (h *PayHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "redirect_token")
	ctx := r.Context()

	userUUID := userIDFromCtx(r)
	if userUUID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	tx, err := h.svc.ConfirmTransaction(ctx, token, userUUID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "PAYMENT_ERROR", err.Error(), nil)
		return
	}

	if tx.Status == "failed" {
		respondError(w, http.StatusBadRequest, "INSUFFICIENT_FUNDS", "Insufficient balance", nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"redirect_url": tx.ReturnURL + "?transaction_id=" + tx.ID.String(),
	})
}

func (h *PayHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "redirect_token")
	ctx := r.Context()

	tx, err := h.svc.GetTransactionByToken(ctx, token)
	if err != nil || tx == nil {
		h.renderer.Execute(w, "error.html", PayPageData{ErrorMessage: "Transaction not found"})
		return
	}

	if tx.CancelURL != "" {
		http.Redirect(w, r, tx.CancelURL, http.StatusSeeOther)
		return
	}

	h.renderer.Execute(w, "error.html", PayPageData{ErrorMessage: "Transaction cancelled"})
}

func formatAmount(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	dollars := amount / 100
	cents := amount % 100
	if negative {
		return "-$" + formatIntToString(dollars) + "." + formatIntToString(cents)
	}
	return "$" + formatIntToString(dollars) + "." + formatIntToString(cents)
}

func formatIntToString(n int64) string {
	if n == 0 {
		return "00"
	}
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	if n < 100 {
		return string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}