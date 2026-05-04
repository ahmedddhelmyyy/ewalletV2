package model

import "errors"

// Sentinel errors used across service and repository layers.
// Handlers map these to the appropriate HTTP status codes and error codes.
var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrTokenNotFound       = errors.New("refresh token not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrBillNotFound        = errors.New("bill not found")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrSelfTransfer        = errors.New("cannot transfer to your own wallet")
	ErrBillAlreadyPaid     = errors.New("bill is already paid")
	ErrCannotDeletePaid    = errors.New("cannot delete a paid bill")
	ErrForbidden           = errors.New("access forbidden")
)
