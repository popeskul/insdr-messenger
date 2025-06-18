package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/popeskul/insdr-messenger/internal/middleware"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func TestRequestID(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID in context")
		}

		// Check if request ID is in response header
		if w.Header().Get(middleware.RequestIDHeader) == "" {
			t.Error("Expected request ID in response header")
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestRateLimiter(t *testing.T) {
	// Create rate limiter with 1 request per second
	rl := middleware.NewRateLimiter(rate.Limit(1), 1)

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Second immediate request should be rate limited
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	// Wait and try again
	time.Sleep(time.Second)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after waiting, got %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	config := middleware.DefaultCORSConfig()
	handler := middleware.CORS(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for preflight, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Expected CORS origin header")
	}
}

func TestRecovery(t *testing.T) {
	logger := zap.NewNop()

	handler := middleware.Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic, got %d", w.Code)
	}
}
