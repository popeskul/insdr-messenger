package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/popeskul/insdr-messenger/internal/api"
	"github.com/popeskul/insdr-messenger/internal/handler"
	"github.com/popeskul/insdr-messenger/internal/middleware"
	"github.com/popeskul/insdr-messenger/internal/scheduler"
	"github.com/popeskul/insdr-messenger/internal/service"
	"github.com/popeskul/insdr-messenger/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestHandler_StartScheduler(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockSchedulerService)
		expectedStatus int
		expectedBody   func(*testing.T, []byte)
	}{
		{
			name: "success",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Start().Return(nil)
			}, expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.SchedulerResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, api.SchedulerResponseStatusStarted, resp.Status)
				assert.Equal(t, "Scheduler started successfully", resp.Message)
			},
		},
		{
			name: "scheduler already running",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Start().Return(scheduler.ErrSchedulerAlreadyRunning)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.ErrorResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, "SCHEDULER_ALREADY_RUNNING", resp.Error)
				assert.Equal(t, "Scheduler is already running", resp.Message)
			},
		},
		{
			name: "internal error",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Start().Return(errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.ErrorResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, middleware.ErrorCodeInternal, resp.Error)
				assert.Equal(t, "Failed to start scheduler", resp.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockScheduler := mocks.NewMockSchedulerService(ctrl)
			tt.setupMocks(mockScheduler)

			svc := &service.Service{
				Scheduler: mockScheduler,
			}

			h := handler.NewHandler(svc, zap.NewNop())

			req := httptest.NewRequest(http.MethodPost, "/scheduler/start", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id"))
			w := httptest.NewRecorder()

			h.StartScheduler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w.Body.Bytes())
		})
	}
}

func TestHandler_StopScheduler(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockSchedulerService)
		expectedStatus int
		expectedBody   func(*testing.T, []byte)
	}{
		{
			name: "success",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Stop().Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.SchedulerResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, api.SchedulerResponseStatusStopped, resp.Status)
				assert.Equal(t, "Scheduler stopped successfully", resp.Message)
			},
		},
		{
			name: "scheduler not running",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Stop().Return(scheduler.ErrSchedulerNotRunning)
			}, expectedStatus: http.StatusConflict,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.ErrorResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, "SCHEDULER_NOT_RUNNING", resp.Error)
				assert.Equal(t, "Scheduler is not running", resp.Message)
			},
		},
		{
			name: "internal error",
			setupMocks: func(m *mocks.MockSchedulerService) {
				m.EXPECT().Stop().Return(errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.ErrorResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, middleware.ErrorCodeInternal, resp.Error)
				assert.Equal(t, "Failed to stop scheduler", resp.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockScheduler := mocks.NewMockSchedulerService(ctrl)
			tt.setupMocks(mockScheduler)

			svc := &service.Service{
				Scheduler: mockScheduler,
			}

			h := handler.NewHandler(svc, zap.NewNop())

			req := httptest.NewRequest(http.MethodPost, "/scheduler/stop", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id"))
			w := httptest.NewRecorder()

			h.StopScheduler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w.Body.Bytes())
		})
	}
}

func TestHandler_GetSentMessages(t *testing.T) {
	tests := []struct {
		name           string
		params         api.GetSentMessagesParams
		setupMocks     func(*mocks.MockMessageService)
		expectedStatus int
		expectedBody   func(*testing.T, []byte)
	}{
		{
			name:   "success with defaults",
			params: api.GetSentMessagesParams{}, setupMocks: func(m *mocks.MockMessageService) {
				m.EXPECT().GetSentMessages(1, 20).Return(&api.MessageListResponse{
					Messages: []api.Message{
						{
							Id:          1,
							PhoneNumber: "+1234567890",
							Content:     ptr("Test message"),
							Status:      api.MessageStatus("sent"),
							SentAt:      ptr(time.Now()),
						},
					},
					Pagination: api.Pagination{
						CurrentPage:  1,
						ItemsPerPage: 20,
						TotalItems:   1,
						TotalPages:   1,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.MessageListResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Messages, 1)
				assert.Equal(t, 1, resp.Pagination.CurrentPage)
				assert.Equal(t, 20, resp.Pagination.ItemsPerPage)
			},
		},
		{
			name: "success with custom params",
			params: api.GetSentMessagesParams{
				Page:  ptr(2),
				Limit: ptr(50),
			},
			setupMocks: func(m *mocks.MockMessageService) {
				m.EXPECT().GetSentMessages(2, 50).Return(&api.MessageListResponse{
					Messages: []api.Message{},
					Pagination: api.Pagination{
						CurrentPage:  2,
						ItemsPerPage: 50,
						TotalItems:   0,
						TotalPages:   0,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.MessageListResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Messages, 0)
				assert.Equal(t, 2, resp.Pagination.CurrentPage)
				assert.Equal(t, 50, resp.Pagination.ItemsPerPage)
			},
		},
		{
			name:   "internal error",
			params: api.GetSentMessagesParams{}, setupMocks: func(m *mocks.MockMessageService) {
				m.EXPECT().GetSentMessages(1, 20).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.ErrorResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, middleware.ErrorCodeInternal, resp.Error)
				assert.Equal(t, "Failed to retrieve sent messages", resp.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMessage := mocks.NewMockMessageService(ctrl)
			tt.setupMocks(mockMessage)

			svc := &service.Service{
				Message: mockMessage,
			}

			h := handler.NewHandler(svc, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/messages", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id"))
			w := httptest.NewRecorder()

			h.GetSentMessages(w, req, tt.params)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w.Body.Bytes())
		})
	}
}

func TestHandler_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockHealthService)
		expectedStatus int
		expectedBody   func(*testing.T, []byte)
	}{
		{
			name: "healthy status",
			setupMocks: func(m *mocks.MockHealthService) {
				m.EXPECT().GetHealth().Return(&service.HealthStatus{
					Status:               api.Healthy,
					SchedulerStatus:      api.HealthResponseSchedulerStatus("running"),
					DatabaseStatus:       api.HealthResponseDatabaseStatus("connected"),
					RedisStatus:          api.HealthResponseRedisStatus("connected"),
					CircuitBreakerStatus: "closed",
					CircuitBreakerState:  api.HealthResponseCircuitBreakerState("closed"),
				})
			},
			expectedStatus: http.StatusOK, expectedBody: func(t *testing.T, body []byte) {
				var resp api.HealthResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, api.Healthy, resp.Status)
				assert.Equal(t, api.HealthResponseSchedulerStatus("running"), *resp.SchedulerStatus)
				assert.Equal(t, api.HealthResponseDatabaseStatus("connected"), *resp.DatabaseStatus)
				assert.Equal(t, api.HealthResponseRedisStatus("connected"), *resp.RedisStatus)
				assert.Equal(t, "closed", *resp.CircuitBreakerStatus)
			},
		},
		{
			name: "unhealthy status",
			setupMocks: func(m *mocks.MockHealthService) {
				m.EXPECT().GetHealth().Return(&service.HealthStatus{
					Status:               api.Unhealthy,
					SchedulerStatus:      api.HealthResponseSchedulerStatus("stopped"),
					DatabaseStatus:       api.HealthResponseDatabaseStatus("disconnected"),
					RedisStatus:          api.HealthResponseRedisStatus("disconnected"),
					CircuitBreakerStatus: "open",
					CircuitBreakerState:  api.HealthResponseCircuitBreakerState("open"),
				})
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.HealthResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, api.Unhealthy, resp.Status)
			},
		},
		{
			name: "degraded status",
			setupMocks: func(m *mocks.MockHealthService) {
				m.EXPECT().GetHealth().Return(&service.HealthStatus{
					Status:               api.Degraded,
					SchedulerStatus:      api.HealthResponseSchedulerStatus("running"),
					DatabaseStatus:       api.HealthResponseDatabaseStatus("connected"),
					RedisStatus:          api.HealthResponseRedisStatus("connected"),
					CircuitBreakerStatus: "open",
					CircuitBreakerState:  api.HealthResponseCircuitBreakerState("open"),
				})
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body []byte) {
				var resp api.HealthResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, api.Degraded, resp.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHealth := mocks.NewMockHealthService(ctrl)
			tt.setupMocks(mockHealth)

			svc := &service.Service{
				Health: mockHealth,
			}

			h := handler.NewHandler(svc, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			h.HealthCheck(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w.Body.Bytes())
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
