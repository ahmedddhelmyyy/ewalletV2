package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ewallet/model"
	"github.com/ewallet/repository"
	"gorm.io/gorm"
)

type transactionService struct {
	db          *gorm.DB
	walletRepo  repository.WalletRepository
	txRepo      repository.TransactionRepository
	userRepo    repository.UserRepository
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(
	db *gorm.DB,
	walletRepo repository.WalletRepository,
	txRepo repository.TransactionRepository,
	userRepo repository.UserRepository,
) TransactionService {
	return &transactionService{
		db:         db,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		userRepo:   userRepo,
	}
}

// Send transfers money from the caller's wallet to a recipient wallet atomically.
func (s *transactionService) Send(userID uuid.UUID, req model.SendRequest) (*model.SendResponse, error) {
	// Load sender wallet
	senderWallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Load recipient wallet
	recipientWallet, err := s.walletRepo.FindByWalletNumber(req.RecipientWalletNumber)
	if err != nil {
		return nil, err
	}

	// Guard: cannot send to own wallet
	if senderWallet.ID == recipientWallet.ID {
		return nil, model.ErrSelfTransfer
	}

	// Guard: sufficient funds
	if senderWallet.Balance < req.Amount {
		return nil, model.ErrInsufficientFunds
	}

	// Resolve category default
	category := req.Category
	if category == "" {
		category = model.CategoryTransfer
	}

	// Load names for denormalisation
	senderUser, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	recipientUser, err := s.userRepo.FindByID(recipientWallet.UserID)
	if err != nil {
		return nil, err
	}

	newSenderBalance := senderWallet.Balance - req.Amount
	newRecipientBalance := recipientWallet.Balance + req.Amount

	var sendTx model.Transaction

	// Atomic: debit sender, credit recipient, write both transaction records
	err = s.db.Transaction(func(tx *gorm.DB) error {
		wRepo := s.walletRepo.WithTx(tx)
		tRepo := s.txRepo.WithTx(tx)

		if err := wRepo.UpdateBalance(senderWallet.ID, newSenderBalance); err != nil {
			return fmt.Errorf("debit sender: %w", err)
		}
		if err := wRepo.UpdateBalance(recipientWallet.ID, newRecipientBalance); err != nil {
			return fmt.Errorf("credit recipient: %w", err)
		}

		// Sender's transaction record
		recipientName := recipientUser.FullName
		sendTx = model.Transaction{
			SenderWalletID:        &senderWallet.ID,
			RecipientWalletID:     &recipientWallet.ID,
			SenderWalletNumber:    &senderWallet.WalletNumber,
			RecipientWalletNumber: &recipientWallet.WalletNumber,
			CounterpartName:       &recipientName,
			Type:                  model.TxTypeSend,
			Status:                model.TxStatusCompleted,
			Amount:                req.Amount,
			Currency:              model.DefaultCurrency,
			Category:              category,
			Note:                  req.Note,
		}
		if err := tRepo.Create(&sendTx); err != nil {
			return fmt.Errorf("create send tx: %w", err)
		}

		// Recipient's transaction record (mirrored receive)
		senderName := senderUser.FullName
		receiveTx := model.Transaction{
			SenderWalletID:        &senderWallet.ID,
			RecipientWalletID:     &recipientWallet.ID,
			SenderWalletNumber:    &senderWallet.WalletNumber,
			RecipientWalletNumber: &recipientWallet.WalletNumber,
			CounterpartName:       &senderName,
			Type:                  model.TxTypeReceive,
			Status:                model.TxStatusCompleted,
			Amount:                req.Amount,
			Currency:              model.DefaultCurrency,
			Category:              category,
			Note:                  req.Note,
		}
		return tRepo.Create(&receiveTx)
	})
	if err != nil {
		return nil, err
	}

	return &model.SendResponse{
		Transaction: mapTransactionToResponse(&sendTx),
		NewBalance:  newSenderBalance,
	}, nil
}

// TopUp credits the caller's wallet with the requested amount (simulated deposit).
func (s *transactionService) TopUp(userID uuid.UUID, req model.TopUpRequest) (*model.TopUpResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	newBalance := wallet.Balance + req.Amount

	var newTx model.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		wRepo := s.walletRepo.WithTx(tx)
		tRepo := s.txRepo.WithTx(tx)

		if err := wRepo.UpdateBalance(wallet.ID, newBalance); err != nil {
			return err
		}

		newTx = model.Transaction{
			RecipientWalletID:     &wallet.ID,
			RecipientWalletNumber: &wallet.WalletNumber,
			Type:                  model.TxTypeTopUp,
			Status:                model.TxStatusCompleted,
			Amount:                req.Amount,
			Currency:              model.DefaultCurrency,
			Category:              model.CategoryTopUp,
			Note:                  req.Note,
		}
		return tRepo.Create(&newTx)
	})
	if err != nil {
		return nil, err
	}

	return &model.TopUpResponse{
		Transaction: mapTransactionToResponse(&newTx),
		NewBalance:  newBalance,
	}, nil
}

// Withdraw debits the caller's wallet by the requested amount (simulated bank withdrawal).
func (s *transactionService) Withdraw(userID uuid.UUID, req model.WithdrawRequest) (*model.WithdrawResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	if wallet.Balance < req.Amount {
		return nil, model.ErrInsufficientFunds
	}

	newBalance := wallet.Balance - req.Amount

	var newTx model.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		wRepo := s.walletRepo.WithTx(tx)
		tRepo := s.txRepo.WithTx(tx)

		if err := wRepo.UpdateBalance(wallet.ID, newBalance); err != nil {
			return err
		}

		newTx = model.Transaction{
			SenderWalletID:     &wallet.ID,
			SenderWalletNumber: &wallet.WalletNumber,
			Type:               model.TxTypeWithdrawal,
			Status:             model.TxStatusCompleted,
			Amount:             req.Amount,
			Currency:           model.DefaultCurrency,
			Category:           model.CategoryWithdrawal,
			Note:               req.Note,
		}
		return tRepo.Create(&newTx)
	})
	if err != nil {
		return nil, err
	}

	return &model.WithdrawResponse{
		Transaction: mapTransactionToResponse(&newTx),
		NewBalance:  newBalance,
	}, nil
}

// GetHistory returns paginated transaction history for the caller's wallet.
func (s *transactionService) GetHistory(userID uuid.UUID, filters model.TransactionFilters) ([]model.TransactionResponse, int64, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, 0, err
	}

	txs, total, err := s.txRepo.FindByWallet(wallet.ID, filters)
	if err != nil {
		return nil, 0, err
	}

	result := make([]model.TransactionResponse, len(txs))
	for i, tx := range txs {
		result[i] = mapTransactionToResponse(&tx)
	}
	return result, total, nil
}

// GetByID returns a single transaction, verifying the caller is a party to it.
func (s *transactionService) GetByID(userID uuid.UUID, transactionID uuid.UUID) (*model.TransactionResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	tx, err := s.txRepo.FindByID(transactionID)
	if err != nil {
		return nil, err
	}

	// Ensure the caller's wallet is either the sender or recipient
	isSender := tx.SenderWalletID != nil && *tx.SenderWalletID == wallet.ID
	isRecipient := tx.RecipientWalletID != nil && *tx.RecipientWalletID == wallet.ID
	if !isSender && !isRecipient {
		return nil, model.ErrForbidden
	}

	resp := mapTransactionToResponse(tx)
	return &resp, nil
}

// mapTransactionToResponse converts a Transaction model to the API response DTO.
func mapTransactionToResponse(tx *model.Transaction) model.TransactionResponse {
	return model.TransactionResponse{
		ID:                    tx.ID,
		Type:                  tx.Type,
		Status:                tx.Status,
		Amount:                tx.Amount,
		Currency:              tx.Currency,
		Category:              tx.Category,
		Note:                  tx.Note,
		SenderWalletNumber:    tx.SenderWalletNumber,
		RecipientWalletNumber: tx.RecipientWalletNumber,
		CounterpartName:       tx.CounterpartName,
		CreatedAt:             tx.CreatedAt,
	}
}
