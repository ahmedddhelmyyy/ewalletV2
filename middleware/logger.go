package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter is a wrapper that captures the HTTP status code written by handlers.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Logger is a middleware that logs each request: method, path, status, duration, and size.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		log.Printf(
			"%s %s %d %dB %s",
			r.Method,
			r.RequestURI,
			wrapped.status,
			wrapped.size,
			time.Since(start).Round(time.Microsecond),
		)
	})
}
