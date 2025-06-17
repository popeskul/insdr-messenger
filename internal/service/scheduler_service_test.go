package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ppopeskul/insider-messenger/internal/config"
	"github.com/ppopeskul/insider-messenger/internal/service"
	"github.com/ppopeskul/insider-messenger/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestSchedulerService_Start_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called (scheduler might call it immediately)
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1, // 1 minute interval
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start scheduler
	err := schedulerService.Start()
	assert.NoError(t, err)

	// Verify it's running
	assert.True(t, schedulerService.IsRunning())

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify it's stopped
	assert.False(t, schedulerService.IsRunning())
}

func TestSchedulerService_Start_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start scheduler
	err := schedulerService.Start()
	require.NoError(t, err)
	assert.True(t, schedulerService.IsRunning())

	// Try to start again - should fail
	err = schedulerService.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Clean up
	_ = schedulerService.Stop()
}

func TestSchedulerService_Stop_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start scheduler
	err := schedulerService.Start()
	require.NoError(t, err)

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify it's stopped
	assert.False(t, schedulerService.IsRunning())
}

func TestSchedulerService_Stop_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Try to stop without starting - should fail
	err := schedulerService.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSchedulerService_IsRunning_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Initially should not be running
	assert.False(t, schedulerService.IsRunning())

	// Start scheduler
	err := schedulerService.Start()
	require.NoError(t, err)

	// Should be running now
	assert.True(t, schedulerService.IsRunning())

	// Stop scheduler
	err = schedulerService.Stop()
	require.NoError(t, err)

	// Should not be running anymore
	assert.False(t, schedulerService.IsRunning())
}

func TestSchedulerService_IsRunning_Failure(t *testing.T) {
	// This test verifies the behavior when checking running status
	// in various failure scenarios
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Should return false when not started
	assert.False(t, schedulerService.IsRunning())

	// Start and immediately stop
	err := schedulerService.Start()
	require.NoError(t, err)
	err = schedulerService.Stop()
	require.NoError(t, err)

	// Should return false after stopping
	assert.False(t, schedulerService.IsRunning())
}

func TestSchedulerService_ExecuteTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Set up expectation - the task should call SendPendingMessages
	callCount := 0
	mockMessageService.EXPECT().
		SendPendingMessages().
		DoAndReturn(func() error {
			callCount++
			return nil
		}).
		MinTimes(1)

	// Create config with very short interval for testing
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1, // 1 minute minimum
		},
	}

	// Create scheduler service with 1 second interval for testing
	logger := zap.NewNop()
	
	// We need to create a custom scheduler for this test
	// since we want to test the task execution
	svc := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start the scheduler
	err := svc.Start()
	require.NoError(t, err)

	// Wait for at least one execution
	time.Sleep(100 * time.Millisecond)

	// Stop the scheduler
	err = svc.Stop()
	require.NoError(t, err)

	// Verify that SendPendingMessages was called at least once
	assert.GreaterOrEqual(t, callCount, 1)
}

func TestSchedulerService_ExecuteTaskError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Set up expectation - the task should handle errors gracefully
	mockMessageService.EXPECT().
		SendPendingMessages().
		Return(errors.New("send error")).
		MinTimes(1)

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1, // Minimum interval
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start the scheduler
	err := schedulerService.Start()
	require.NoError(t, err)

	// Wait for at least one execution
	time.Sleep(100 * time.Millisecond)

	// Scheduler should still be running despite task errors
	assert.True(t, schedulerService.IsRunning())

	// Stop the scheduler
	err = schedulerService.Stop()
	require.NoError(t, err)
}

func TestSchedulerService_MultipleStartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Multiple start/stop cycles
	for i := 0; i < 3; i++ {
		// Start
		err := schedulerService.Start()
		require.NoError(t, err)
		assert.True(t, schedulerService.IsRunning())

		// Stop
		err = schedulerService.Stop()
		require.NoError(t, err)
		assert.False(t, schedulerService.IsRunning())
	}
}

func TestSchedulerService_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock message service
	mockMessageService := mocks.NewMockMessageService(ctrl)

	// Expect SendPendingMessages to be called
	mockMessageService.EXPECT().SendPendingMessages().Return(nil).AnyTimes()

	// Create config
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			IntervalMinutes: 1,
		},
	}

	// Create scheduler service
	logger := zap.NewNop()
	schedulerService := service.NewSchedulerService(cfg, mockMessageService, logger)

	// Start scheduler
	err := schedulerService.Start()
	require.NoError(t, err)

	// Run concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Check IsRunning multiple times
	go func() {
		for i := 0; i < 10; i++ {
			_ = schedulerService.IsRunning()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Try to start multiple times (should fail)
	go func() {
		for i := 0; i < 5; i++ {
			_ = schedulerService.Start()
			time.Sleep(20 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Check status
	go func() {
		for i := 0; i < 10; i++ {
			_ = schedulerService.IsRunning()
			time.Sleep(15 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)
}
