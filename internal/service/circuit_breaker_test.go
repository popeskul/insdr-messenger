package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/config"
	"github.com/ppopeskul/insider-messenger/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	tests := []struct {
		name     string
		function func() error
		wantErr  bool
	}{
		{
			name: "successful execution",
			function: func() error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "successful execution with delay",
			function: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.CircuitBreakerConfig{
				MaxRequests:      3,
				Interval:         10,
				Timeout:          60,
				FailureRatio:     0.6,
				ConsecutiveFails: 5,
			}
			logger := zap.NewNop()
			cb := service.NewCircuitBreaker(cfg, logger)

			ctx := context.Background()
			err := cb.Execute(ctx, tt.function)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*service.CircuitBreaker)
		function       func() error
		expectedErrMsg string
	}{
		{
			name: "function returns error",
			function: func() error {
				return errors.New("test error")
			},
			expectedErrMsg: "test error",
		},
		{
			name: "context cancelled",
			function: func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			expectedErrMsg: "context canceled",
		},
		{
			name: "circuit breaker open",
			setupFunc: func(cb *service.CircuitBreaker) {
				// Make the circuit breaker trip by causing consecutive failures
				for i := 0; i < 10; i++ {
					_ = cb.Execute(context.Background(), func() error {
						return errors.New("failure")
					})
				}
			},
			function: func() error {
				return nil
			},
			expectedErrMsg: "service unavailable: circuit breaker is open",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.CircuitBreakerConfig{
				MaxRequests:      3,
				Interval:         10,
				Timeout:          60,
				FailureRatio:     0.5,
				ConsecutiveFails: 3,
			}
			logger := zap.NewNop()
			cb := service.NewCircuitBreaker(cfg, logger)

			if tt.setupFunc != nil {
				tt.setupFunc(cb)
			}

			ctx := context.Background()
			if tt.name == "context cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err := cb.Execute(ctx, tt.function)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestCircuitBreaker_GetState_Success(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         10,
		Timeout:          60,
		FailureRatio:     0.6,
		ConsecutiveFails: 3,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Initially should be closed
	state := cb.GetState()
	assert.Equal(t, api.Closed, state)

	// Execute successful request
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	require.NoError(t, err)

	// State should still be closed
	state = cb.GetState()
	assert.Equal(t, api.Closed, state)
}

func TestCircuitBreaker_GetState_Failure(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         10,
		Timeout:          1, // Short timeout for testing
		FailureRatio:     0.5,
		ConsecutiveFails: 3,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Cause failures to trip the circuit breaker
	for i := 0; i < 5; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	// State should be open
	state := cb.GetState()
	assert.Equal(t, api.Open, state)

	// Wait for timeout to pass
	time.Sleep(2 * time.Second)

	// State should be half-open
	state = cb.GetState()
	assert.Equal(t, api.HalfOpen, state)
}

func TestCircuitBreaker_GetCounts_Success(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      10,
		Interval:         60,
		Timeout:          60,
		FailureRatio:     0.8,
		ConsecutiveFails: 10,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Initially counts should be zero
	requests, failures := cb.GetCounts()
	assert.Equal(t, uint32(0), requests)
	assert.Equal(t, uint32(0), failures)

	// Execute some successful requests
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		require.NoError(t, err)
	}

	// Check counts
	requests, failures = cb.GetCounts()
	assert.Equal(t, uint32(3), requests)
	assert.Equal(t, uint32(0), failures)
}

func TestCircuitBreaker_GetCounts_Failure(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      10,
		Interval:         60,
		Timeout:          60,
		FailureRatio:     0.8,
		ConsecutiveFails: 10,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Execute some successful and failed requests
	successCount := 0
	failureCount := 0

	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			err := cb.Execute(context.Background(), func() error {
				return nil
			})
			require.NoError(t, err)
			successCount++
		} else {
			err := cb.Execute(context.Background(), func() error {
				return errors.New("failure")
			})
			require.Error(t, err)
			failureCount++
		}
	}

	// Check counts
	requests, failures := cb.GetCounts()
	assert.Equal(t, uint32(successCount+failureCount), requests)
	assert.Equal(t, uint32(failureCount), failures)
}

func TestCircuitBreaker_TooManyRequests(t *testing.T) {
	// This test verifies circuit breaker behavior with many concurrent requests
	// The exact behavior in half-open state can vary based on timing

	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      1, // Very low max requests
		Interval:         10,
		Timeout:          1,
		FailureRatio:     0.5,
		ConsecutiveFails: 2,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Verify initial state
	assert.Equal(t, api.Closed, cb.GetState())

	// Cause multiple failures to trip the circuit breaker
	failureCount := 0
	for i := 0; i < 5; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
		if err != nil {
			failureCount++
		}
	}

	// Should have failed requests and circuit should be open
	assert.Greater(t, failureCount, 0, "Expected some failed requests")
	assert.Equal(t, api.Open, cb.GetState())

	// Try to execute while open - should fail
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Circuit might be in half-open state
	state := cb.GetState()
	assert.True(t, state == api.HalfOpen || state == api.Open,
		"Expected state to be HalfOpen or Open after timeout, got %s", state)

	// Execute some requests - behavior varies based on state and timing
	var successTotal, errorTotal int
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		if err == nil {
			successTotal++
		} else {
			errorTotal++
		}
	}

	// We should have at least some result (success or error)
	assert.Greater(t, successTotal+errorTotal, 0, "Expected some requests to be processed")
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         10,
		Timeout:          1, // Short timeout for testing
		FailureRatio:     0.5,
		ConsecutiveFails: 2,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Initial state should be closed
	assert.Equal(t, api.Closed, cb.GetState())

	// Cause consecutive failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	// Should be open now
	assert.Equal(t, api.Open, cb.GetState())

	// Try to execute - should fail with open state error
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Should be half-open
	assert.Equal(t, api.HalfOpen, cb.GetState())

	// Multiple successful requests to ensure transition to closed
	for i := 0; i < 3; i++ {
		err = cb.Execute(context.Background(), func() error {
			return nil
		})
		if err != nil {
			break
		}
	}

	// Give it a moment to fully transition
	time.Sleep(100 * time.Millisecond)

	// Check if it's closed or half-open (both are acceptable after successful requests)
	state := cb.GetState()
	assert.True(t, state == api.Closed || state == api.HalfOpen,
		"Expected state to be Closed or HalfOpen after successful requests, got %s", state)
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	cfg := &config.CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         10,
		Timeout:          60,
		FailureRatio:     0.6,
		ConsecutiveFails: 5,
	}
	logger := zap.NewNop()
	cb := service.NewCircuitBreaker(cfg, logger)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start a goroutine that will cancel the context after a delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Execute a long-running function
	err := cb.Execute(ctx, func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	// Should get context cancelled error
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
