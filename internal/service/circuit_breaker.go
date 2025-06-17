// Package service provides business logic implementation for the application.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/config"
)

type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	logger *zap.Logger
}

func NewCircuitBreaker(cfg *config.CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        "webhook-circuit-breaker",
		MaxRequests: cfg.MaxRequests,
		Interval:    time.Duration(cfg.Interval) * time.Second,
		Timeout:     time.Duration(cfg.Timeout) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= cfg.ConsecutiveFails && failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}

	return &CircuitBreaker{
		cb:     gobreaker.NewCircuitBreaker(settings),
		logger: logger,
	}
}

// Execute runs the given function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	_, err := cb.cb.Execute(func() (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, fn()
		}
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			cb.logger.Warn("Circuit breaker is open, request blocked")
			return fmt.Errorf("service unavailable: circuit breaker is open")
		}
		if errors.Is(err, gobreaker.ErrTooManyRequests) {
			cb.logger.Warn("Circuit breaker: too many requests")
			return fmt.Errorf("service unavailable: too many requests")
		}
		return err
	}

	return nil
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() api.HealthResponseCircuitBreakerState {
	state := cb.cb.State()
	switch state {
	case gobreaker.StateClosed:
		return api.Closed
	case gobreaker.StateHalfOpen:
		return api.HalfOpen
	case gobreaker.StateOpen:
		return api.Open
	default:
		return api.Closed
	}
}

// GetCounts returns the current counts of the circuit breaker.
func (cb *CircuitBreaker) GetCounts() (requests, failures uint32) {
	counts := cb.cb.Counts()
	return counts.Requests, counts.TotalFailures
}
