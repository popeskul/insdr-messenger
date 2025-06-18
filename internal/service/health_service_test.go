package service_test

import (
	"errors"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/popeskul/insdr-messenger/internal/api"
	"github.com/popeskul/insdr-messenger/internal/repository/mocks"
	"github.com/popeskul/insdr-messenger/internal/service"
	servicemocks "github.com/popeskul/insdr-messenger/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHealthService_GetHealth_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockRepo := mocks.NewMockRepository(ctrl)
	mockScheduler := servicemocks.NewMockSchedulerService(ctrl)
	mockMessage := servicemocks.NewMockMessageService(ctrl)

	// Mock Redis client - we'll use a real client pointing to a non-existent server
	// This will simulate a disconnected state
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent Redis server
	})

	// Set up expectations
	mockScheduler.EXPECT().IsRunning().Return(true)
	mockRepo.EXPECT().Ping().Return(nil)
	mockMessage.EXPECT().GetCircuitBreakerStatus().Return(api.Closed, uint32(100), uint32(5))

	// Create health service
	healthService := service.NewHealthService(mockRepo, redisClient, mockScheduler, mockMessage)

	// Get health status
	status := healthService.GetHealth()

	// Assertions
	require.NotNil(t, status)
	assert.Equal(t, api.Unhealthy, status.Status) // Unhealthy because Redis is disconnected
	assert.Equal(t, api.HealthResponseSchedulerStatusRunning, status.SchedulerStatus)
	assert.Equal(t, api.HealthResponseDatabaseStatusConnected, status.DatabaseStatus)
	assert.Equal(t, api.HealthResponseRedisStatusDisconnected, status.RedisStatus)
	assert.Equal(t, api.Closed, status.CircuitBreakerState)
	assert.Equal(t, "Requests: 100, Failures: 5 (5.0%)", status.CircuitBreakerStatus)
}

func TestHealthService_GetHealth_Failure(t *testing.T) {
	tests := []struct {
		name                    string
		setupMocks              func(*mocks.MockRepository, *servicemocks.MockSchedulerService, *servicemocks.MockMessageService)
		expectedStatus          api.HealthResponseStatus
		expectedSchedulerStatus api.HealthResponseSchedulerStatus
		expectedDatabaseStatus  api.HealthResponseDatabaseStatus
		expectedCBState         api.HealthResponseCircuitBreakerState
	}{
		{
			name: "scheduler stopped, database connected, circuit breaker closed",
			setupMocks: func(repo *mocks.MockRepository, scheduler *servicemocks.MockSchedulerService, message *servicemocks.MockMessageService) {
				scheduler.EXPECT().IsRunning().Return(false)
				repo.EXPECT().Ping().Return(nil)
				message.EXPECT().GetCircuitBreakerStatus().Return(api.Closed, uint32(50), uint32(10))
			},
			expectedStatus:          api.Unhealthy, // Redis disconnected
			expectedSchedulerStatus: api.HealthResponseSchedulerStatusStopped,
			expectedDatabaseStatus:  api.HealthResponseDatabaseStatusConnected,
			expectedCBState:         api.Closed,
		},
		{
			name: "database disconnected",
			setupMocks: func(repo *mocks.MockRepository, scheduler *servicemocks.MockSchedulerService, message *servicemocks.MockMessageService) {
				scheduler.EXPECT().IsRunning().Return(true)
				repo.EXPECT().Ping().Return(errors.New("connection failed"))
				message.EXPECT().GetCircuitBreakerStatus().Return(api.Closed, uint32(0), uint32(0))
			},
			expectedStatus:          api.Unhealthy,
			expectedSchedulerStatus: api.HealthResponseSchedulerStatusRunning,
			expectedDatabaseStatus:  api.HealthResponseDatabaseStatusDisconnected,
			expectedCBState:         api.Closed,
		},
		{
			name: "circuit breaker open",
			setupMocks: func(repo *mocks.MockRepository, scheduler *servicemocks.MockSchedulerService, message *servicemocks.MockMessageService) {
				scheduler.EXPECT().IsRunning().Return(true)
				repo.EXPECT().Ping().Return(nil)
				message.EXPECT().GetCircuitBreakerStatus().Return(api.Open, uint32(100), uint32(60))
			},
			expectedStatus:          api.Degraded, // Open circuit breaker means degraded
			expectedSchedulerStatus: api.HealthResponseSchedulerStatusRunning,
			expectedDatabaseStatus:  api.HealthResponseDatabaseStatusConnected,
			expectedCBState:         api.Open,
		},
		{
			name: "everything failing",
			setupMocks: func(repo *mocks.MockRepository, scheduler *servicemocks.MockSchedulerService, message *servicemocks.MockMessageService) {
				scheduler.EXPECT().IsRunning().Return(false)
				repo.EXPECT().Ping().Return(errors.New("db error"))
				message.EXPECT().GetCircuitBreakerStatus().Return(api.Open, uint32(1000), uint32(999))
			},
			expectedStatus:          api.Degraded, // DB disconnected + open CB = degraded (CB takes precedence)
			expectedSchedulerStatus: api.HealthResponseSchedulerStatusStopped,
			expectedDatabaseStatus:  api.HealthResponseDatabaseStatusDisconnected,
			expectedCBState:         api.Open,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mocks
			mockRepo := mocks.NewMockRepository(ctrl)
			mockScheduler := servicemocks.NewMockSchedulerService(ctrl)
			mockMessage := servicemocks.NewMockMessageService(ctrl)

			// Mock Redis client - disconnected
			redisClient := redis.NewClient(&redis.Options{
				Addr: "localhost:9999",
			})

			// Set up expectations
			tt.setupMocks(mockRepo, mockScheduler, mockMessage)

			// Create health service
			healthService := service.NewHealthService(mockRepo, redisClient, mockScheduler, mockMessage)

			// Get health status
			status := healthService.GetHealth()

			// Assertions
			require.NotNil(t, status)
			assert.Equal(t, tt.expectedStatus, status.Status)
			assert.Equal(t, tt.expectedSchedulerStatus, status.SchedulerStatus)
			assert.Equal(t, tt.expectedDatabaseStatus, status.DatabaseStatus)
			assert.Equal(t, api.HealthResponseRedisStatusDisconnected, status.RedisStatus)
			assert.Equal(t, tt.expectedCBState, status.CircuitBreakerState)
		})
	}
}

func TestHealthService_CircuitBreakerStatusFormatting(t *testing.T) {
	tests := []struct {
		name             string
		requests         uint32
		failures         uint32
		expectedCBStatus string
	}{
		{
			name:             "no requests",
			requests:         0,
			failures:         0,
			expectedCBStatus: "No requests yet",
		},
		{
			name:             "all successful",
			requests:         100,
			failures:         0,
			expectedCBStatus: "Requests: 100, Failures: 0 (0.0%)",
		},
		{
			name:             "some failures",
			requests:         100,
			failures:         25,
			expectedCBStatus: "Requests: 100, Failures: 25 (25.0%)",
		},
		{
			name:             "all failures",
			requests:         50,
			failures:         50,
			expectedCBStatus: "Requests: 50, Failures: 50 (100.0%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mocks
			mockRepo := mocks.NewMockRepository(ctrl)
			mockScheduler := servicemocks.NewMockSchedulerService(ctrl)
			mockMessage := servicemocks.NewMockMessageService(ctrl)

			redisClient := redis.NewClient(&redis.Options{
				Addr: "localhost:9999",
			})

			// Set up expectations
			mockScheduler.EXPECT().IsRunning().Return(true)
			mockRepo.EXPECT().Ping().Return(nil)
			mockMessage.EXPECT().GetCircuitBreakerStatus().Return(api.Closed, tt.requests, tt.failures)

			// Create health service
			healthService := service.NewHealthService(mockRepo, redisClient, mockScheduler, mockMessage)

			// Get health status
			status := healthService.GetHealth()

			// Assert circuit breaker status formatting
			assert.Equal(t, tt.expectedCBStatus, status.CircuitBreakerStatus)
		})
	}
}
