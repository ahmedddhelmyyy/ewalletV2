package model

type Merchant struct {
	ID            string
	APIKey        string
	Secret        string
	WebhookURL    string
	WebhookSecret string
}