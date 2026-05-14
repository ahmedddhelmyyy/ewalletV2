package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ewallet/model"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type MockWalletService struct {
	userBalances    map[string]int64
	merchantBalances map[string]int64
}

func NewMockWalletService() *MockWalletService {
	return &MockWalletService{
		userBalances: map[string]int64{
			"test_user": 1000000,
		},
		merchantBalances: map[string]int64{
			"merch_123": 0,
		},
	}
}

func (s *MockWalletService) GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	return &model.Wallet{
		ID:      uuid.New(),
		Balance:  s.userBalances[userID],
		Currency: "USD",
	}, nil
}

func (s *MockWalletService) GetMerchantBalance(ctx context.Context, merchantID string) (int64, error) {
	return s.merchantBalances[merchantID], nil
}

func (s *MockWalletService) CreditMerchant(ctx context.Context, merchantID string, amount int64) error {
	s.merchantBalances[merchantID] += amount
	return nil
}

func (s *MockWalletService) DebitMerchant(ctx context.Context, merchantID string, amount int64) error {
	balance := s.merchantBalances[merchantID]
	if balance < amount {
		return ErrInsufficientFunds
	}
	s.merchantBalances[merchantID] = balance - amount
	return nil
}

func (s *MockWalletService) Deduct(ctx context.Context, userID string, amount int64) error {
	balance := s.userBalances[userID]
	if balance < amount {
		return ErrInsufficientFunds
	}
	s.userBalances[userID] = balance - amount
	return nil
}