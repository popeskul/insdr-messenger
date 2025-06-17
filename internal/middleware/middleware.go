package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Config holds middleware configuration.
type Config struct {
	Logger *zap.Logger

	CORS *CORSConfig

	RateLimit      rate.Limit
	RateLimitBurst int

	RequestTimeout time.Duration
}

// Chain creates a middleware chain with all configured middleware.
func Chain(config *Config) func(http.Handler) http.Handler {
	rateLimiter := NewRateLimiter(config.RateLimit, config.RateLimitBurst)

	return func(handler http.Handler) http.Handler {
		// Apply middleware in order (outer to inner)
		h := handler

		h = Timeout(config.RequestTimeout)(h)

		h = rateLimiter.Middleware()(h)

		if config.CORS != nil {
			h = CORS(config.CORS)(h)
		}

		h = Recovery(config.Logger)(h)

		h = RequestID(h)

		h = Logger(config.Logger)(h)

		return h
	}
}
