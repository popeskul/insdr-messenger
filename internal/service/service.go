package service

import (
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/ppopeskul/insider-messenger/internal/config"
	"github.com/ppopeskul/insider-messenger/internal/repository"
)

type Service struct {
	Message   MessageService
	Scheduler SchedulerService
	Health    HealthService
}

func NewService(
	cfg *config.Config,
	repo repository.Repository,
	redisClient *redis.Client,
	logger *zap.Logger,
) *Service {
	messageService := NewMessageService(cfg, repo, redisClient, logger)
	schedulerService := NewSchedulerService(cfg, messageService, logger)
	healthService := NewHealthService(repo, redisClient, schedulerService, messageService)

	return &Service{
		Message:   messageService,
		Scheduler: schedulerService,
		Health:    healthService,
	}
}
