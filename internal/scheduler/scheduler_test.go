package scheduler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/popeskul/insdr-messenger/internal/scheduler"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestScheduler_Start(t *testing.T) {
	tests := []struct {
		name           string
		setupScheduler func() *scheduler.Scheduler
		expectedError  error
	}{
		{
			name: "success",
			setupScheduler: func() *scheduler.Scheduler {
				taskFunc := func(ctx context.Context) error {
					return nil
				}
				return scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, taskFunc)
			},
			expectedError: nil,
		},
		{
			name: "already running", setupScheduler: func() *scheduler.Scheduler {
				s := scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, func(ctx context.Context) error {
					return nil
				})
				err := s.Start(context.Background())
				assert.NoError(t, err)
				return s
			},
			expectedError: scheduler.ErrSchedulerAlreadyRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setupScheduler()
			defer func() {
				if s.IsRunning() {
					_ = s.Stop()
				}
			}()

			err := s.Start(context.Background())
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestScheduler_Stop(t *testing.T) {
	tests := []struct {
		name           string
		setupScheduler func() *scheduler.Scheduler
		expectedError  error
	}{
		{
			name: "success",
			setupScheduler: func() *scheduler.Scheduler {
				s := scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, func(ctx context.Context) error {
					return nil
				})
				err := s.Start(context.Background())
				assert.NoError(t, err)
				return s
			},
			expectedError: nil,
		},
		{
			name: "not running",
			setupScheduler: func() *scheduler.Scheduler {
				return scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, func(ctx context.Context) error {
					return nil
				})
			},
			expectedError: scheduler.ErrSchedulerNotRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setupScheduler()
			err := s.Stop()
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestScheduler_IsRunning(t *testing.T) {
	tests := []struct {
		name           string
		setupScheduler func() *scheduler.Scheduler
		expected       bool
	}{
		{
			name: "running",
			setupScheduler: func() *scheduler.Scheduler {
				s := scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, func(ctx context.Context) error {
					return nil
				})
				err := s.Start(context.Background())
				assert.NoError(t, err)
				return s
			},
			expected: true,
		},
		{
			name: "not running",
			setupScheduler: func() *scheduler.Scheduler {
				return scheduler.NewScheduler(zap.NewNop(), 100*time.Millisecond, func(ctx context.Context) error {
					return nil
				})
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setupScheduler()
			defer func() {
				if s.IsRunning() {
					_ = s.Stop()
				}
			}()

			assert.Equal(t, tt.expected, s.IsRunning())
		})
	}
}

func TestScheduler_TaskExecution(t *testing.T) {
	tests := []struct {
		name         string
		taskFunc     func(context.Context) error
		interval     time.Duration
		testDuration time.Duration
		minCalls     int
		maxCalls     int
	}{
		{
			name: "task executes multiple times",
			taskFunc: func(ctx context.Context) error {
				return nil
			},
			interval:     50 * time.Millisecond,
			testDuration: 250 * time.Millisecond,
			minCalls:     5,
			maxCalls:     7,
		},
		{
			name: "task handles errors",
			taskFunc: func(ctx context.Context) error {
				return errors.New("task error")
			},
			interval:     50 * time.Millisecond,
			testDuration: 150 * time.Millisecond,
			minCalls:     3,
			maxCalls:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			taskFunc := func(ctx context.Context) error {
				callCount++
				return tt.taskFunc(ctx)
			}

			s := scheduler.NewScheduler(zap.NewNop(), tt.interval, taskFunc)
			err := s.Start(context.Background())
			assert.NoError(t, err)
			time.Sleep(tt.testDuration)

			err = s.Stop()
			assert.NoError(t, err)

			assert.GreaterOrEqual(t, callCount, tt.minCalls)
			assert.LessOrEqual(t, callCount, tt.maxCalls)
		})
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	var mu sync.Mutex
	taskCalls := 0
	taskFunc := func(ctx context.Context) error {
		mu.Lock()
		taskCalls++
		mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := scheduler.NewScheduler(zap.NewNop(), 50*time.Millisecond, taskFunc)

	err := s.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, s.IsRunning())

	// Wait for at least 2 executions
	time.Sleep(120 * time.Millisecond)

	mu.Lock()
	callsBeforeCancel := taskCalls
	mu.Unlock()

	// Should have at least 2 calls (initial + 2 intervals)
	assert.GreaterOrEqual(t, callsBeforeCancel, 2)

	cancel()

	// Wait for scheduler to stop
	time.Sleep(100 * time.Millisecond)
	assert.False(t, s.IsRunning())

	// Get final call count
	mu.Lock()
	finalCalls := taskCalls
	mu.Unlock()

	// Should not have significantly more calls after cancel
	assert.LessOrEqual(t, finalCalls-callsBeforeCancel, 1)
}

func TestScheduler_ConcurrentAccess(t *testing.T) {
	s := scheduler.NewScheduler(zap.NewNop(), 50*time.Millisecond, func(ctx context.Context) error {
		return nil
	})

	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		go func() {
			if err := s.Start(context.Background()); err != nil && err != scheduler.ErrSchedulerAlreadyRunning {
				errors <- err
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	assert.True(t, s.IsRunning())
	assert.Len(t, errors, 0)

	err := s.Stop()
	assert.NoError(t, err)
}
