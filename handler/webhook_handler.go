package handler

// import (
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"errors"
// 	"io"
// 	"net/http"

// 	"github.com/ewallet/model"
// 	"github.com/ewallet/service"
// )

// type WebhookHandler struct {
// 	webhookSvc    service.WebhookService
// 	signingSecret string // shared secret from the external service
// }

// func NewWebhookHandler(svc service.WebhookService, signingSecret string) *WebhookHandler {
// 	return &WebhookHandler{webhookSvc: svc, signingSecret: signingSecret}
// }

// // POST /api/v1/webhooks/transfers
// func (h *WebhookHandler) HandleIncomingTransfer(w http.ResponseWriter, r *http.Request) {
// 	// 1. Read raw body — needed for signature verification before parsing
// 	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB cap
// 	if err != nil {
// 		respondError(w, http.StatusBadRequest, "cannot read body")
// 		return
// 	}
// 	defer r.Body.Close()

// 	// 2. Verify HMAC-SHA256 signature sent by the external service
// 	if !h.validSignature(body, r.Header.Get("X-Signature-SHA256")) {
// 		respondError(w, http.StatusUnauthorized, "invalid signature")
// 		return
// 	}

// 	// 3. Parse payload
// 	var payload model.IncomingTransferPayload
// 	if err := json.Unmarshal(body, &payload); err != nil {
// 		respondError(w, http.StatusBadRequest, "malformed payload")
// 		return
// 	}

// 	// 4. Validate required fields
// 	if payload.ExternalID == "" || payload.Amount <= 0 || payload.SenderNumber == "" {
// 		respondError(w, http.StatusUnprocessableEntity, "missing required fields: external_id, amount, sender_number")
// 		return
// 	}

// 	// 5. Resolve which wallet receives this — options:
// 	//    a) payload contains target wallet number → look it up
// 	//    b) single-tenant: use a fixed system wallet ID from config
// 	//    Here we use (a) — assume payload has "recipient_wallet_number"
// 	walletID, err := h.webhookSvc.ResolveWalletByNumber(r.Context(), payload.SenderNumber)
// 	// (see note below on routing)

// 	// 6. Process
// 	err = h.webhookSvc.ProcessIncomingTransfer(r.Context(), walletID, payload)
// 	if errors.Is(err, service.ErrDuplicateWebhook) {
// 		// 200 OK — idempotent, safe to acknowledge
// 		respondJSON(w, http.StatusOK, map[string]string{"status": "already_processed"})
// 		return
// 	}
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "failed to process transfer")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
// }

// // validSignature computes HMAC-SHA256 of the raw body and compares to the
// // signature header sent by the external service.
// func (h *WebhookHandler) validSignature(body []byte, receivedSig string) bool {
// 	if h.signingSecret == "" {
// 		// If no secret configured, skip verification (dev mode only — warn loudly)
// 		return true
// 	}
// 	mac := hmac.New(sha256.New, []byte(h.signingSecret))
// 	mac.Write(body)
// 	expected := hex.EncodeToString(mac.Sum(nil))
// 	return hmac.Equal([]byte(expected), []byte(receivedSig))
// }