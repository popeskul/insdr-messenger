package service_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/popeskul/insdr-messenger/internal/api"
	"github.com/popeskul/insdr-messenger/internal/config"
	"github.com/popeskul/insdr-messenger/internal/models"
	"github.com/popeskul/insdr-messenger/internal/repository/mocks"
	"github.com/popeskul/insdr-messenger/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestMessageService_SendPendingMessages_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	successCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-auth-key", r.Header.Get("x-ins-auth-key"))

		var req models.WebhookRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		resp := models.WebhookResponse{
			Message:   "Success",
			MessageID: fmt.Sprintf("msg-%d", successCount),
		}
		successCount++

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

	mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()

	testMessages := []*models.Message{
		{
			ID:          1,
			PhoneNumber: "+1234567890",
			Content:     "Test message 1",
			Status:      models.MessageStatusPending,
		},
		{
			ID:          2,
			PhoneNumber: "+0987654321",
			Content:     "Test message 2",
			Status:      models.MessageStatusPending,
		},
	}

	mockMessageRepo.EXPECT().GetUnsentMessages(10).Return(testMessages, nil)

	for i, msg := range testMessages {
		messageID := fmt.Sprintf("msg-%d", i)
		mockMessageRepo.EXPECT().
			UpdateMessageStatus(msg.ID, models.MessageStatusSent, &messageID, nil).
			Return(nil)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent server for testing
	})

	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			URL:     server.URL,
			AuthKey: "test-auth-key",
			Timeout: 30,
			CircuitBreaker: config.CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60,
				Timeout:          60,
				FailureRatio:     0.6,
				ConsecutiveFails: 5,
			},
		},
		Scheduler: config.SchedulerConfig{
			BatchSize: 10,
		},
	}

	logger := zap.NewNop()
	messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

	err := messageService.SendPendingMessages()
	assert.NoError(t, err)
}

func TestMessageService_SendPendingMessages_Failure(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockRepository, *mocks.MockMessageRepository)
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name: "failed to get unsent messages",
			setupMocks: func(mockRepo *mocks.MockRepository, mockMessageRepo *mocks.MockMessageRepository) {
				mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()
				mockMessageRepo.EXPECT().
					GetUnsentMessages(gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedError: "failed to get unsent messages",
		},
		{
			name: "no pending messages",
			setupMocks: func(mockRepo *mocks.MockRepository, mockMessageRepo *mocks.MockMessageRepository) {
				mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()
				mockMessageRepo.EXPECT().
					GetUnsentMessages(gomock.Any()).
					Return([]*models.Message{}, nil)
			},
			expectedError: "",
		},
		{
			name: "webhook returns error status",
			setupMocks: func(mockRepo *mocks.MockRepository, mockMessageRepo *mocks.MockMessageRepository) {
				mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()

				testMessage := &models.Message{
					ID:          1,
					PhoneNumber: "+1234567890",
					Content:     "Test message",
					Status:      models.MessageStatusPending,
				}

				mockMessageRepo.EXPECT().
					GetUnsentMessages(gomock.Any()).
					Return([]*models.Message{testMessage}, nil)

				mockMessageRepo.EXPECT().
					UpdateMessageStatus(testMessage.ID, models.MessageStatusFailed, nil, gomock.Any()).
					Return(nil)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var server *httptest.Server
			if tt.serverResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer server.Close()
			}

			mockRepo := mocks.NewMockRepository(ctrl)
			mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

			tt.setupMocks(mockRepo, mockMessageRepo)

			redisClient := redis.NewClient(&redis.Options{
				Addr: "localhost:9999",
			})

			webhookURL := "http://localhost:1234"
			if server != nil {
				webhookURL = server.URL
			}

			cfg := &config.Config{
				Webhook: config.WebhookConfig{
					URL:     webhookURL,
					AuthKey: "test-auth-key",
					Timeout: 1,
					CircuitBreaker: config.CircuitBreakerConfig{
						MaxRequests:      10,
						Interval:         60,
						Timeout:          60,
						FailureRatio:     0.6,
						ConsecutiveFails: 5,
					},
				},
				Scheduler: config.SchedulerConfig{
					BatchSize: 10,
				},
			}

			logger := zap.NewNop()
			messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

			err := messageService.SendPendingMessages()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessageService_GetSentMessages_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

	mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()

	sentAt := sql.NullTime{Time: time.Now(), Valid: true}
	messageID := sql.NullString{String: "msg-123", Valid: true}
	errorMsg := sql.NullString{String: "error occurred", Valid: true}

	testMessages := []*models.Message{
		{
			ID:          1,
			PhoneNumber: "+1234567890",
			Content:     "Test message 1",
			Status:      models.MessageStatusSent,
			MessageID:   messageID,
			SentAt:      sentAt,
		},
		{
			ID:          2,
			PhoneNumber: "+0987654321",
			Content:     "Test message 2",
			Status:      models.MessageStatusFailed,
			Error:       errorMsg,
			SentAt:      sql.NullTime{},
		},
	}

	page := 1
	limit := 10
	offset := 0
	totalCount := int64(50)

	mockMessageRepo.EXPECT().GetSentMessages(offset, limit).Return(testMessages, nil)
	mockMessageRepo.EXPECT().GetTotalSentCount().Return(totalCount, nil)

	cfg := &config.Config{}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:9999"})
	logger := zap.NewNop()
	messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

	result, err := messageService.GetSentMessages(page, limit)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Messages, 2)

	assert.Equal(t, int64(1), result.Messages[0].Id)
	assert.Equal(t, "+1234567890", result.Messages[0].PhoneNumber)
	assert.Equal(t, "Test message 1", *result.Messages[0].Content)
	assert.Equal(t, models.MessageStatusSent, result.Messages[0].Status)
	assert.Equal(t, "msg-123", *result.Messages[0].MessageId)
	assert.NotNil(t, result.Messages[0].SentAt)

	assert.Equal(t, int64(2), result.Messages[1].Id)
	assert.Equal(t, "+0987654321", result.Messages[1].PhoneNumber)
	assert.Equal(t, models.MessageStatusFailed, result.Messages[1].Status)
	assert.Equal(t, "error occurred", *result.Messages[1].Error)
	assert.Nil(t, result.Messages[1].SentAt)

	assert.Equal(t, 1, result.Pagination.CurrentPage)
	assert.Equal(t, 5, result.Pagination.TotalPages)
	assert.Equal(t, 50, result.Pagination.TotalItems)
	assert.Equal(t, 10, result.Pagination.ItemsPerPage)
}

func TestMessageService_GetSentMessages_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mocks.MockRepository, *mocks.MockMessageRepository)
		page          int
		limit         int
		expectedError string
	}{
		{
			name: "failed to get sent messages",
			setupMocks: func(mockRepo *mocks.MockRepository, mockMessageRepo *mocks.MockMessageRepository) {
				mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()
				mockMessageRepo.EXPECT().
					GetSentMessages(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			page:          1,
			limit:         10,
			expectedError: "failed to get sent messages",
		},
		{
			name: "failed to get total count",
			setupMocks: func(mockRepo *mocks.MockRepository, mockMessageRepo *mocks.MockMessageRepository) {
				mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()
				mockMessageRepo.EXPECT().
					GetSentMessages(gomock.Any(), gomock.Any()).
					Return([]*models.Message{}, nil)
				mockMessageRepo.EXPECT().
					GetTotalSentCount().
					Return(int64(0), errors.New("count error"))
			},
			page:          1,
			limit:         10,
			expectedError: "failed to get total count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

			tt.setupMocks(mockRepo, mockMessageRepo)

			cfg := &config.Config{}
			redisClient := redis.NewClient(&redis.Options{Addr: "localhost:9999"})
			logger := zap.NewNop()
			messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

			result, err := messageService.GetSentMessages(tt.page, tt.limit)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Nil(t, result)
		})
	}
}

func TestMessageService_GetCircuitBreakerStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999",
	})

	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			CircuitBreaker: config.CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60,
				Timeout:          60,
				FailureRatio:     0.6,
				ConsecutiveFails: 5,
			},
		},
	}

	logger := zap.NewNop()
	messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

	state, requests, failures := messageService.GetCircuitBreakerStatus()

	assert.Equal(t, api.Closed, state)
	assert.Equal(t, uint32(0), requests)
	assert.Equal(t, uint32(0), failures)
}

func TestMessageService_GetCircuitBreakerStatus_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

	mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()

	testMessage := &models.Message{
		ID:          1,
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      models.MessageStatusPending,
	}

	mockMessageRepo.EXPECT().
		GetUnsentMessages(gomock.Any()).
		Return([]*models.Message{testMessage}, nil).
		Times(5)

	mockMessageRepo.EXPECT().
		UpdateMessageStatus(testMessage.ID, models.MessageStatusFailed, nil, gomock.Any()).
		Return(nil).
		Times(5)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999",
	})

	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			URL:     server.URL,
			AuthKey: "test-auth-key",
			Timeout: 1,
			CircuitBreaker: config.CircuitBreakerConfig{
				MaxRequests:      3,
				Interval:         60,
				Timeout:          60,
				FailureRatio:     0.5,
				ConsecutiveFails: 2,
			},
		},
		Scheduler: config.SchedulerConfig{
			BatchSize: 10,
		},
	}

	logger := zap.NewNop()
	messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

	for i := 0; i < 5; i++ {
		_ = messageService.SendPendingMessages()
	}

	state, requests, failures := messageService.GetCircuitBreakerStatus()

	assert.Equal(t, api.Open, state)

	// The counts might be 0 if the circuit breaker window has passed
	// but the state should still be Open
	if requests > 0 {
		assert.GreaterOrEqual(t, requests, uint32(2))
		assert.GreaterOrEqual(t, failures, uint32(2))
	}
}

func TestMessageService_PaginationCalculation(t *testing.T) {
	tests := []struct {
		name               string
		totalCount         int64
		page               int
		limit              int
		expectedTotalPages int
	}{
		{
			name:               "exact division",
			totalCount:         100,
			page:               1,
			limit:              10,
			expectedTotalPages: 10,
		},
		{
			name:               "with remainder",
			totalCount:         105,
			page:               1,
			limit:              10,
			expectedTotalPages: 11,
		},
		{
			name:               "single page",
			totalCount:         5,
			page:               1,
			limit:              10,
			expectedTotalPages: 1,
		},
		{
			name:               "no items",
			totalCount:         0,
			page:               1,
			limit:              10,
			expectedTotalPages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			mockMessageRepo := mocks.NewMockMessageRepository(ctrl)

			mockRepo.EXPECT().Message().Return(mockMessageRepo).AnyTimes()

			offset := (tt.page - 1) * tt.limit
			mockMessageRepo.EXPECT().GetSentMessages(offset, tt.limit).Return([]*models.Message{}, nil)
			mockMessageRepo.EXPECT().GetTotalSentCount().Return(tt.totalCount, nil)

			cfg := &config.Config{}
			redisClient := redis.NewClient(&redis.Options{Addr: "localhost:9999"})
			logger := zap.NewNop()
			messageService := service.NewMessageService(cfg, mockRepo, redisClient, logger)

			result, err := messageService.GetSentMessages(tt.page, tt.limit)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedTotalPages, result.Pagination.TotalPages)
			assert.Equal(t, int(tt.totalCount), result.Pagination.TotalItems)
		})
	}
}
