package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

// Timeout middleware adds a timeout to requests.
func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Channel to track if handler completes
			done := make(chan struct{})

			// Run handler in goroutine
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					w.WriteHeader(http.StatusRequestTimeout)
					render.JSON(w, r, map[string]interface{}{
						"error":   ErrorCodeRequestTimeout,
						"message": ErrorMessageRequestTimeout,
					})
				}
			}
		})
	}
}
