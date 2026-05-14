package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ewallet/model"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type MockWalletService struct {
	balances map[string]int64
}

func NewMockWalletService() *MockWalletService {
	return &MockWalletService{
		balances: map[string]int64{
			"test_user": 1000000, // $10,000 in cents
		},
	}
}

func (s *MockWalletService) GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	return &model.Wallet{
		ID:      uuid.New(),
		Balance:  s.balances[userID],
		Currency: "USD",
	}, nil
}

func (s *MockWalletService) Deduct(ctx context.Context, userID string, amount int64) error {
	balance := s.balances[userID]
	if balance < amount {
		return ErrInsufficientFunds
	}
	s.balances[userID] = balance - amount
	return nil
}