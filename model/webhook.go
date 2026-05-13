package model

import "time"

// IncomingTransferPayload is the shape the external service POSTs to us.
// Map every field they send — ignore extras via json tags.
type IncomingTransferPayload struct {
	ExternalID    string    `json:"external_id"`    // their transaction ID (idempotency key)
	SenderNumber  string    `json:"sender_number"`  // wallet number on their side
	Amount        int64     `json:"amount"`         // in cents — confirm with the external service
	Currency      string    `json:"currency"`
	Note          *string   `json:"note"`
	SenderName    *string   `json:"sender_name"`
	TransferredAt time.Time `json:"transferred_at"`
}