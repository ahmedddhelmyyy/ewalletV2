package service

import (
	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
)

type walletService struct {
	walletRepo repository.WalletRepository
}

// NewWalletService creates a new WalletService.
func NewWalletService(walletRepo repository.WalletRepository) WalletService {
	return &walletService{walletRepo: walletRepo}
}

// GetWalletByUserID loads the authenticated user's wallet and formats it for the API response.
func (s *walletService) GetWalletByUserID(userID uuid.UUID) (*model.WalletResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	var ownerResp *model.WalletOwnerResponse
	if wallet.User != nil {
		ownerResp = &model.WalletOwnerResponse{
			ID:       wallet.User.ID,
			FullName: wallet.User.FullName,
			Email:    wallet.User.Email,
		}
	}

	resp := mapWalletToResponse(wallet, ownerResp)
	return &resp, nil
}
