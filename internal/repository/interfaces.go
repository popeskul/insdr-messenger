package repository

import "github.com/ppopeskul/insider-messenger/internal/models"

// Repository interface defines all repository operations.
type Repository interface {
	// Ping checks database connectivity
	Ping() error

	// Message returns message repository
	Message() MessageRepository
}

// MessageRepository interface defines message operations.
type MessageRepository interface {
	GetUnsentMessages(limit int) ([]*models.Message, error)
	UpdateMessageStatus(id int64, status models.MessageStatus, messageID *string, errorMsg *string) error
	GetSentMessages(offset, limit int) ([]*models.Message, error)
	GetTotalSentCount() (int64, error)
	CreateMessage(phoneNumber, content string) error
}
