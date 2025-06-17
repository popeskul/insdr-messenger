package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ppopeskul/insider-messenger/internal/api"
	"github.com/ppopeskul/insider-messenger/internal/models"
)

type messageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) MessageRepository {
	return &messageRepository{
		db: db,
	}
}

// GetUnsentMessages retrieves unsent messages from the database.
func (r *messageRepository) GetUnsentMessages(limit int) ([]*models.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, error, created_at, sent_at, updated_at
		FROM messages
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	var messages []*models.Message
	err := r.db.Select(&messages, query, models.MessageStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsent messages: %w", err)
	}

	return messages, nil
}

// UpdateMessageStatus updates the status of a message.
func (r *messageRepository) UpdateMessageStatus(id int64, status api.MessageStatus, messageID *string, errorMsg *string) error {
	query := `
		UPDATE messages
		SET status = $2, 
		    message_id = $3, 
		    error = $4, 
		    sent_at = $5,
		    updated_at = $6
		WHERE id = $1
	`

	var sentAt sql.NullTime
	if status == models.MessageStatusSent {
		sentAt = sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		}
	}

	var msgID sql.NullString
	if messageID != nil {
		msgID = sql.NullString{
			String: *messageID,
			Valid:  true,
		}
	}

	var errMsg sql.NullString
	if errorMsg != nil {
		errMsg = sql.NullString{
			String: *errorMsg,
			Valid:  true,
		}
	}

	_, err := r.db.Exec(query, id, status, msgID, errMsg, sentAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	return nil
}

// GetSentMessages retrieves sent messages with pagination.
func (r *messageRepository) GetSentMessages(offset, limit int) ([]*models.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, error, created_at, sent_at, updated_at
		FROM messages
		WHERE status = $1
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3	`

	var messages []*models.Message
	err := r.db.Select(&messages, query, models.MessageStatusSent, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent messages: %w", err)
	}

	return messages, nil
}

// GetTotalSentCount returns the total count of sent messages.
func (r *messageRepository) GetTotalSentCount() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM messages WHERE status = $1`

	err := r.db.Get(&count, query, models.MessageStatusSent)
	if err != nil {
		return 0, fmt.Errorf("failed to get total sent count: %w", err)
	}

	return count, nil
}

// CreateMessage creates a new message in the database.
func (r *messageRepository) CreateMessage(phoneNumber, content string) error {
	query := `
		INSERT INTO messages (phone_number, content, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)	`

	now := time.Now()
	_, err := r.db.Exec(query, phoneNumber, content, models.MessageStatusPending, now, now)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}
