// Package models defines data structures used throughout the application.
package models

import (
	"database/sql"
	"time"

	"github.com/popeskul/insdr-messenger/internal/api"
)

type MessageStatus = api.MessageStatus

const (
	MessageStatusPending = api.Pending
	MessageStatusSent    = api.Sent
	MessageStatusFailed  = api.Failed
)

// Message represents a message in the database.
type Message struct {
	ID          int64          `db:"id" json:"id"`
	PhoneNumber string         `db:"phone_number" json:"phone_number"`
	Content     string         `db:"content" json:"content"`
	Status      MessageStatus  `db:"status" json:"status"`
	MessageID   sql.NullString `db:"message_id" json:"message_id,omitempty"`
	Error       sql.NullString `db:"error" json:"error,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	SentAt      sql.NullTime   `db:"sent_at" json:"sent_at,omitempty"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

type WebhookRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type WebhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}
