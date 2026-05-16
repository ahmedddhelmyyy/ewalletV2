package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ewallet/middleware"
	"github.com/ewallet/model"
	"github.com/ewallet/service"
)

type MerchantHandler struct {
	svc service.MerchantServiceInterface
}

func NewMerchantHandler(svc service.MerchantServiceInterface) *MerchantHandler {
	return &MerchantHandler{svc: svc}
}

const basePaymentURL = "http://localhost:8080"

func (h *MerchantHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	merchant := middleware.GetMerchantFromCtx(r)
	if merchant.ID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req model.CreateMerchantTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed.", map[string]string{"error": err.Error()})
		return
	}

	ctx := r.Context()
	tx, created, err := h.svc.CreateTransaction(ctx, merchant.ID, merchant.UserID, &req, r.Header.Get("Idempotency-Key"))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	redirectURL := basePaymentURL + "/pay/" + tx.RedirectToken
	expiresAt := tx.ExpiresAt.Format(time.RFC3339)

	if !created {
		respondSuccess(w, http.StatusOK, model.MerchantTransactionResponse{
			TransactionID: tx.ID.String(),
			RedirectURL:   redirectURL,
			Status:        tx.Status,
			ExpiresAt:     expiresAt,
		})
		return
	}

	respondSuccess(w, http.StatusCreated, model.MerchantTransactionResponse{
		TransactionID: tx.ID.String(),
		RedirectURL:   redirectURL,
		Status:        tx.Status,
		ExpiresAt:     expiresAt,
	})
}

func (h *MerchantHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	merchant := middleware.GetMerchantFromCtx(r)
	if merchant.ID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	txID := chi.URLParam(r, "transaction_id")
	ctx := r.Context()

	tx, err := h.svc.GetTransactionByID(ctx, txID, merchant.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, tx)
}

func (h *MerchantHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	merchant := middleware.GetMerchantFromCtx(r)
	if merchant.ID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	merchantID := chi.URLParam(r, "merchant_id")
	if merchantID != merchant.ID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	ctx := r.Context()
	balance, err := h.svc.GetMerchantBalance(ctx, merchantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]interface{}{
		"merchant_id": merchantID,
		"balance":     balance,
		"currency":    "USD",
	})
}
