package service

import "github.com/ppopeskul/insider-messenger/internal/api"

type HealthStatus struct {
	Status               api.HealthResponseStatus              `json:"status"`
	SchedulerStatus      api.HealthResponseSchedulerStatus     `json:"scheduler_status"`
	DatabaseStatus       api.HealthResponseDatabaseStatus      `json:"database_status"`
	RedisStatus          api.HealthResponseRedisStatus         `json:"redis_status"`
	CircuitBreakerStatus string                                `json:"circuit_breaker_status,omitempty"`
	CircuitBreakerState  api.HealthResponseCircuitBreakerState `json:"circuit_breaker_state,omitempty"`
}
