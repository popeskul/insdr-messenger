package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/repository"
)

type healthService struct {
	repo             repository.Repository
	redisClient      *redis.Client
	schedulerService SchedulerService
	messageService   MessageService
}

func NewHealthService(
	repo repository.Repository,
	redisClient *redis.Client,
	schedulerService SchedulerService,
	messageService MessageService,
) HealthService {
	return &healthService{
		repo:             repo,
		redisClient:      redisClient,
		schedulerService: schedulerService,
		messageService:   messageService,
	}
}

func (s *healthService) GetHealth() *HealthStatus {
	status := &HealthStatus{
		Status: api.Healthy,
	}

	if s.schedulerService.IsRunning() {
		status.SchedulerStatus = api.HealthResponseSchedulerStatusRunning
	} else {
		status.SchedulerStatus = api.HealthResponseSchedulerStatusStopped
	}

	status.DatabaseStatus = s.checkDatabaseHealth()

	status.RedisStatus = s.checkRedisHealth()

	state, requests, failures := s.messageService.GetCircuitBreakerStatus()
	status.CircuitBreakerState = state
	if requests > 0 {
		failureRate := float64(failures) / float64(requests) * 100
		status.CircuitBreakerStatus = fmt.Sprintf("Requests: %d, Failures: %d (%.1f%%)", requests, failures, failureRate)
	} else {
		status.CircuitBreakerStatus = "No requests yet"
	}

	// Determine overall health
	if status.DatabaseStatus != api.HealthResponseDatabaseStatusConnected || status.RedisStatus != api.HealthResponseRedisStatusConnected {
		status.Status = api.Unhealthy
	}

	// If circuit breaker is open, set status to degraded
	if state == api.Open {
		status.Status = api.Degraded
	}

	return status
}

func (s *healthService) checkDatabaseHealth() api.HealthResponseDatabaseStatus {
	err := s.repo.Ping()
	if err != nil {
		return api.HealthResponseDatabaseStatusDisconnected
	}
	return api.HealthResponseDatabaseStatusConnected
}

func (s *healthService) checkRedisHealth() api.HealthResponseRedisStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return api.HealthResponseRedisStatusDisconnected
	}

	return api.HealthResponseRedisStatusConnected
}
