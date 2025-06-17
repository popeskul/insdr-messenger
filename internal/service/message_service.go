package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/config"
	"github.com/ppopeskul/insider-messenger/internal/models"
	"github.com/ppopeskul/insider-messenger/internal/repository"
)

type messageService struct {
	cfg            *config.Config
	repo           repository.Repository
	redisClient    *redis.Client
	httpClient     *http.Client
	logger         *zap.Logger
	circuitBreaker *CircuitBreaker
}

func NewMessageService(
	cfg *config.Config,
	repo repository.Repository,
	redisClient *redis.Client,
	logger *zap.Logger,
) MessageService {
	cb := NewCircuitBreaker(&cfg.Webhook.CircuitBreaker, logger)

	return &messageService{
		cfg:         cfg,
		repo:        repo,
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Webhook.Timeout) * time.Second,
		},
		logger:         logger,
		circuitBreaker: cb,
	}
}

// SendPendingMessages sends all pending messages.
func (s *messageService) SendPendingMessages() error {
	s.logger.Info("Starting to send pending messages")

	messages, err := s.repo.Message().GetUnsentMessages(s.cfg.Scheduler.BatchSize)
	if err != nil {
		s.logger.Error("Failed to get unsent messages", zap.Error(err))
		return fmt.Errorf("failed to get unsent messages: %w", err)
	}

	if len(messages) == 0 {
		s.logger.Info("No pending messages to send")
		return nil
	}

	s.logger.Info("Found pending messages", zap.Int("count", len(messages)))

	for _, msg := range messages {
		if err := s.sendMessage(msg); err != nil {
			s.logger.Error("Failed to send message",
				zap.Int64("messageID", msg.ID),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// sendMessage sends a single message
func (s *messageService) sendMessage(msg *models.Message) error {
	// Execute through circuit breaker
	err := s.circuitBreaker.Execute(context.Background(), func() error {
		reqBody := models.WebhookRequest{
			To:      msg.PhoneNumber,
			Content: msg.Content,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", s.cfg.Webhook.URL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-ins-auth-key", s.cfg.Webhook.AuthKey)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				s.logger.Warn("Failed to close response body", zap.Error(err))
			}
		}()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var webhookResp models.WebhookResponse
		if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
				tempMessageID := fmt.Sprintf("temp-%d-%d", msg.ID, time.Now().Unix())
				webhookResp.MessageID = tempMessageID
			} else {
				return fmt.Errorf("failed to decode response: %w", err)
			}
		}

		// Update message status to sent
		if err := s.repo.Message().UpdateMessageStatus(msg.ID, models.MessageStatusSent, &webhookResp.MessageID, nil); err != nil {
			return fmt.Errorf("failed to update message status: %w", err)
		}

		// Cache message ID in Redis (bonus feature)
		ctx := context.Background()
		cacheKey := fmt.Sprintf("message:%s", webhookResp.MessageID)
		cacheValue := fmt.Sprintf("%d:%s", msg.ID, time.Now().Format(time.RFC3339))

		if err := s.redisClient.Set(ctx, cacheKey, cacheValue, 24*time.Hour).Err(); err != nil {
			s.logger.Warn("Failed to cache message ID in Redis",
				zap.String("messageID", webhookResp.MessageID),
				zap.Error(err))
		}

		s.logger.Info("Message sent successfully",
			zap.Int64("messageID", msg.ID),
			zap.String("externalMessageID", webhookResp.MessageID),
			zap.String("circuitBreakerState", string(s.circuitBreaker.GetState())))

		return nil
	})

	// Handle circuit breaker errors
	if err != nil {
		errMsg := err.Error()
		if updateErr := s.repo.Message().UpdateMessageStatus(msg.ID, models.MessageStatusFailed, nil, &errMsg); updateErr != nil {
			s.logger.Error("Failed to update message status",
				zap.Int64("messageID", msg.ID),
				zap.Error(updateErr))
		}

		requests, failures := s.circuitBreaker.GetCounts()
		s.logger.Error("Failed to send message",
			zap.Int64("messageID", msg.ID),
			zap.Error(err),
			zap.String("circuitBreakerState", string(s.circuitBreaker.GetState())),
			zap.Uint32("totalRequests", requests),
			zap.Uint32("totalFailures", failures))

		return err
	}

	return nil
}

// GetSentMessages retrieves sent messages with pagination.
func (s *messageService) GetSentMessages(page, limit int) (*api.MessageListResponse, error) {
	offset := (page - 1) * limit

	messages, err := s.repo.Message().GetSentMessages(offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent messages: %w", err)
	}

	totalCount, err := s.repo.Message().GetTotalSentCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	totalPages := int(totalCount) / limit
	if int(totalCount)%limit > 0 {
		totalPages++
	}

	var messageResponses []api.Message
	for _, msg := range messages {
		msgResp := api.Message{
			Id:          msg.ID,
			PhoneNumber: msg.PhoneNumber,
			Content:     &msg.Content,
			Status:      msg.Status,
		}

		if msg.SentAt.Valid {
			msgResp.SentAt = &msg.SentAt.Time
		}

		if msg.MessageID.Valid {
			msgResp.MessageId = &msg.MessageID.String
		}

		if msg.Error.Valid {
			msgResp.Error = &msg.Error.String
		}

		messageResponses = append(messageResponses, msgResp)
	}

	return &api.MessageListResponse{
		Messages: messageResponses,
		Pagination: api.Pagination{
			CurrentPage:  page,
			TotalPages:   totalPages,
			TotalItems:   int(totalCount),
			ItemsPerPage: limit,
		},
	}, nil
}

func (s *messageService) GetCircuitBreakerStatus() (state api.HealthResponseCircuitBreakerState, requests uint32, failures uint32) {
	state = s.circuitBreaker.GetState()
	requests, failures = s.circuitBreaker.GetCounts()
	return
}
