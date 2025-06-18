package service

import "github.com/popeskul/insdr-messenger/internal/api"

type MessageService interface {
	SendPendingMessages() error
	GetSentMessages(page, limit int) (*api.MessageListResponse, error)
	GetCircuitBreakerStatus() (state api.HealthResponseCircuitBreakerState, requests uint32, failures uint32)
}

type SchedulerService interface {
	Start() error
	Stop() error
	IsRunning() bool
}

type HealthService interface {
	GetHealth() *HealthStatus
}
