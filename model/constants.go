package model

// Transaction type constants
const (
	TxTypeSend        = "send"
	TxTypeReceive     = "receive"
	TxTypeTopUp       = "top_up"
	TxTypeWithdrawal  = "withdrawal"
	TxTypeBillPayment = "bill_payment"
)

// Transaction status constants
const (
	TxStatusPending   = "pending"
	TxStatusCompleted = "completed"
	TxStatusFailed    = "failed"
)

// Category constants
const (
	CategoryFood       = "food"
	CategoryTransport  = "transport"
	CategoryBills      = "bills"
	CategoryTransfer   = "transfer"
	CategoryTopUp      = "top_up"
	CategoryWithdrawal = "withdrawal"
	CategoryOther      = "other"
)

// Bill status constants
const (
	BillStatusPending = "pending"
	BillStatusPaid    = "paid"
)

// ValidCategories is the set of allowed category values for user-supplied input.
var ValidCategories = map[string]bool{
	CategoryFood:       true,
	CategoryTransport:  true,
	CategoryBills:      true,
	CategoryTransfer:   true,
	CategoryTopUp:      true,
	CategoryWithdrawal: true,
	CategoryOther:      true,
}

// ValidTxTypes is the set of allowed transaction type filter values.
var ValidTxTypes = map[string]bool{
	TxTypeSend:        true,
	TxTypeReceive:     true,
	TxTypeTopUp:       true,
	TxTypeWithdrawal:  true,
	TxTypeBillPayment: true,
}

// DebitTypes are transaction types that reduce the wallet balance.
var DebitTypes = []string{TxTypeSend, TxTypeWithdrawal, TxTypeBillPayment}

// CreditTypes are transaction types that increase the wallet balance.
var CreditTypes = []string{TxTypeReceive, TxTypeTopUp}

// DefaultCurrency is the only currency supported in v1.
const DefaultCurrency = "USD"

// MinTopUpAmount is the minimum top-up in cents ($1.00).
const MinTopUpAmount int64 = 100

// MaxTopUpAmount is the maximum top-up in cents ($100,000.00).
const MaxTopUpAmount int64 = 10_000_000

// MinWithdrawAmount is the minimum withdrawal in cents ($1.00).
const MinWithdrawAmount int64 = 100

// MaxFlowRangeDays is the maximum number of days allowed for the flow endpoint.
const MaxFlowRangeDays = 365
