package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ppopeskul/insider-messenger/internal/config"
	"github.com/ppopeskul/insider-messenger/internal/scheduler"
)

type schedulerService struct {
	scheduler      *scheduler.Scheduler
	messageService MessageService
	logger         *zap.Logger
}

func NewSchedulerService(
	cfg *config.Config,
	messageService MessageService,
	logger *zap.Logger,
) SchedulerService {
	interval := time.Duration(cfg.Scheduler.IntervalMinutes) * time.Minute

	svc := &schedulerService{
		messageService: messageService,
		logger:         logger,
	}

	svc.scheduler = scheduler.NewScheduler(logger, interval, svc.executeSendTask)
	return svc
}

func (s *schedulerService) Start() error {
	ctx := context.Background()
	return s.scheduler.Start(ctx)
}

func (s *schedulerService) Stop() error {
	return s.scheduler.Stop()
}

func (s *schedulerService) IsRunning() bool {
	return s.scheduler.IsRunning()
}

func (s *schedulerService) executeSendTask(_ context.Context) error {
	return s.messageService.SendPendingMessages()
}
