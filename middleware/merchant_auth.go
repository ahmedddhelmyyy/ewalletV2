package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Merchant struct {
	ID            string
	APIKey        string
	Secret        string
	WebhookURL    string
	WebhookSecret string
}

var merchants = map[string]Merchant{
	"key_abc123": {
		ID:            "merch_123",
		APIKey:        "key_abc123",
		Secret:        "supersecret",
		WebhookURL:    "https://shop.com/api/payments/wallet/webhook",
		WebhookSecret: "whsec_test",
	},
}

func AuthenticateMerchant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Api-Key")
		signature := r.Header.Get("X-Signature")
		timestamp := r.Header.Get("X-Timestamp")

		log.Printf("[MERCHANT_AUTH] apiKey=%s, signature=%s, timestamp=%s\n", apiKey, signature, timestamp)

		if apiKey == "" || signature == "" || timestamp == "" {
			log.Printf("[MERCHANT_AUTH] missing headers\n")
			http.Error(w, `{"error":"missing authentication headers"}`, http.StatusUnauthorized)
			return
		}

		merchant, ok := merchants[apiKey]
		if !ok {
			http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid timestamp"}`, http.StatusUnauthorized)
			return
		}

		if time.Now().Unix()-ts > 300 || ts-time.Now().Unix() > 300 {
			http.Error(w, `{"error":"timestamp outside valid window"}`, http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedSig := hmacSHA256(merchant.Secret, string(body)+timestamp)
		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
			return
		}

		ctx := WithMerchant(r.Context(), merchant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hmacSHA256(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func GetMerchantFromCtx(r *http.Request) Merchant {
	merch, ok := r.Context().Value(merchantCtxKey).(Merchant)
	if !ok {
		return Merchant{}
	}
	return merch
}

const merchantCtxKey string = "merchant"

func WithMerchant(ctx context.Context, m Merchant) context.Context {
	return context.WithValue(ctx, merchantCtxKey, m)
}