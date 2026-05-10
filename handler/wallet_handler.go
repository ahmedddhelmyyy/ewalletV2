package handler

import (
	"net/http"

	"github.com/ewallet/service"
)

// WalletHandler holds HTTP handlers for wallet endpoints.
type WalletHandler struct {
	walletSvc service.WalletService
}

// NewWalletHandler creates a new WalletHandler.
func NewWalletHandler(walletSvc service.WalletService) *WalletHandler {
	return &WalletHandler{walletSvc: walletSvc}
}

// GetWallet godoc
//
//	@Summary		Get my wallet
//	@Description	Returns the authenticated user's wallet including current balance in cents. Divide by 100 for display value.
//	@Tags			Wallet
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	SuccessEnvelope{data=model.WalletResponse}
//	@Failure		401	{object}	ErrorEnvelope
//	@Failure		404	{object}	ErrorEnvelope	"Wallet not found"
//	@Router			/wallet [get]
func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r)
	resp, err := h.walletSvc.GetWalletByUserID(userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, resp)
}