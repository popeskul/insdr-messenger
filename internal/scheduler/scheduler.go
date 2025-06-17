package scheduler

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Scheduler manages message sending tasks.
type Scheduler struct {
	logger    *zap.Logger
	interval  time.Duration
	taskFunc  func(context.Context) error
	stopCh    chan struct{}
	doneCh    chan struct{}
	isRunning bool
	mu        sync.RWMutex
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(logger *zap.Logger, interval time.Duration, taskFunc func(context.Context) error) *Scheduler {
	return &Scheduler{
		logger:   logger,
		interval: interval,
		taskFunc: taskFunc,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return ErrSchedulerAlreadyRunning
	}

	s.isRunning = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})

	go s.run(ctx)

	s.logger.Info("Scheduler started", zap.Duration("interval", s.interval))
	return nil
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh

	s.mu.Lock()
	s.isRunning = false
	s.mu.Unlock()

	s.logger.Info("Scheduler stopped")
	return nil
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// run executes the scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	// Execute immediately on start
	if err := s.executeTask(ctx); err != nil {
		s.logger.Error("Failed to execute initial task", zap.Error(err))
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context canceled")
			return
		case <-s.stopCh:
			s.logger.Info("Scheduler stop signal received")
			return
		case <-ticker.C:
			if err := s.executeTask(ctx); err != nil {
				s.logger.Error("Failed to execute scheduled task", zap.Error(err))
			}
		}
	}
}

// executeTask runs the task function with error handling
func (s *Scheduler) executeTask(ctx context.Context) error {
	s.logger.Info("Executing scheduled task")

	taskCtx, cancel := context.WithTimeout(ctx, s.interval-time.Second)
	defer cancel()

	err := s.taskFunc(taskCtx)
	if err != nil {
		s.logger.Error("Task execution failed", zap.Error(err))
	} else {
		s.logger.Info("Task execution completed successfully")
	}
	return err
}
