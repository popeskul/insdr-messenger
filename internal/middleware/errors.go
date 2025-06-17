package middleware

// Common error codes used by middleware
const (
	ErrorCodeInternal          = "INTERNAL_ERROR"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrorCodeRequestTimeout    = "REQUEST_TIMEOUT"
)

// Common error messages used by middleware
const (
	ErrorMessageInternal          = "An internal error occurred"
	ErrorMessageRateLimitExceeded = "Too many requests"
	ErrorMessageRequestTimeout    = "Request timeout"
)
