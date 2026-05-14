package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const ContextKeyUserID string = "userID"

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w)
				return
			}

			// Strip ALL "Bearer " prefixes — handles both single and double prefix
			tokenStr := authHeader
			for strings.HasPrefix(tokenStr, "Bearer ") {
				tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
			}
			tokenStr = strings.TrimSpace(tokenStr)

			if tokenStr == "" {
				respondUnauthorized(w)
				return
			}

			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				respondUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"Missing or invalid access token."}}`))
}