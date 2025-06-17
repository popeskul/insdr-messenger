// Package handler provides HTTP request handlers for the application.
package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"go.uber.org/zap"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/middleware"
	"github.com/ppopeskul/insider-messenger/internal/scheduler"
	"github.com/ppopeskul/insider-messenger/internal/service"
)

const (
	errorCodeSchedulerAlreadyRunning = "SCHEDULER_ALREADY_RUNNING"
	errorCodeSchedulerNotRunning     = "SCHEDULER_NOT_RUNNING"
)

const (
	errorMessageSchedulerAlreadyRunning  = "Scheduler is already running"
	errorMessageSchedulerNotRunning      = "Scheduler is not running"
	errorMessageFailedToStartScheduler   = "Failed to start scheduler"
	errorMessageFailedToStopScheduler    = "Failed to stop scheduler"
	errorMessageFailedToRetrieveMessages = "Failed to retrieve sent messages"
)

const (
	schedulerMessageStarted = "Scheduler started successfully"
	schedulerMessageStopped = "Scheduler stopped successfully"
)

type Handler struct {
	service *service.Service
	logger  *zap.Logger
}

// NewHandler creates a new handler instance that implements api.ServerInterface.
func NewHandler(service *service.Service, logger *zap.Logger) api.ServerInterface {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// StartScheduler implements api.ServerInterface.
func (h *Handler) StartScheduler(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	err := h.service.Scheduler.Start()
	if err != nil {
		if errors.Is(err, scheduler.ErrSchedulerAlreadyRunning) {
			h.sendError(w, r, http.StatusConflict, errorCodeSchedulerAlreadyRunning, errorMessageSchedulerAlreadyRunning)
			return
		}

		h.logger.Error("Failed to start scheduler",
			zap.String("request_id", requestID),
			zap.Error(err))
		h.sendError(w, r, http.StatusInternalServerError, middleware.ErrorCodeInternal, errorMessageFailedToStartScheduler)
		return
	}

	render.JSON(w, r, api.SchedulerResponse{
		Status:  api.SchedulerResponseStatusStarted,
		Message: schedulerMessageStarted,
	})
}

// StopScheduler implements api.ServerInterface.
func (h *Handler) StopScheduler(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())

	err := h.service.Scheduler.Stop()
	if err != nil {
		if errors.Is(err, scheduler.ErrSchedulerNotRunning) {
			h.sendError(w, r, http.StatusConflict, errorCodeSchedulerNotRunning, errorMessageSchedulerNotRunning)
			return
		}

		h.logger.Error("Failed to stop scheduler",
			zap.String("request_id", requestID),
			zap.Error(err))
		h.sendError(w, r, http.StatusInternalServerError, middleware.ErrorCodeInternal, errorMessageFailedToStopScheduler)
		return
	}

	render.JSON(w, r, api.SchedulerResponse{
		Status:  api.SchedulerResponseStatusStopped,
		Message: schedulerMessageStopped,
	})
}

// GetSentMessages implements api.ServerInterface.
func (h *Handler) GetSentMessages(w http.ResponseWriter, r *http.Request, params api.GetSentMessagesParams) {
	page := 1
	limit := 20

	if params.Page != nil && *params.Page >= 1 {
		page = *params.Page
	}

	if params.Limit != nil && *params.Limit >= 1 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	result, err := h.service.Message.GetSentMessages(page, limit)
	if err != nil {
		requestID := middleware.GetRequestID(r.Context())
		h.logger.Error("Failed to get sent messages",
			zap.String("request_id", requestID),
			zap.Error(err))
		h.sendError(w, r, http.StatusInternalServerError, middleware.ErrorCodeInternal, errorMessageFailedToRetrieveMessages)
		return
	}

	render.JSON(w, r, result)
}

// HealthCheck implements api.ServerInterface.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := h.service.Health.GetHealth()

	response := api.HealthResponse{
		Status:    health.Status,
		Timestamp: time.Now(),
	}

	if health.SchedulerStatus != "" {
		status := health.SchedulerStatus
		response.SchedulerStatus = &status
	}

	if health.DatabaseStatus != "" {
		status := health.DatabaseStatus
		response.DatabaseStatus = &status
	}

	if health.RedisStatus != "" {
		status := health.RedisStatus
		response.RedisStatus = &status
	}

	if health.CircuitBreakerStatus != "" {
		response.CircuitBreakerStatus = &health.CircuitBreakerStatus
	}

	if health.CircuitBreakerState != "" {
		state := health.CircuitBreakerState
		response.CircuitBreakerState = &state
	}

	switch health.Status {
	case api.Unhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	case api.Degraded:
		// Return 200 but with degraded status for circuit breaker open state
		// This allows monitoring to detect degraded state while service is still accessible
		response.Status = api.Degraded
	}

	render.JSON(w, r, response)
}

func (h *Handler) sendError(w http.ResponseWriter, r *http.Request, statusCode int, errorCode, message string) {
	render.Status(r, statusCode)
	render.JSON(w, r, api.ErrorResponse{
		Error:   errorCode,
		Message: message,
		Timestamp: func() *time.Time {
			t := time.Now()
			return &t
		}(),
	})
}
