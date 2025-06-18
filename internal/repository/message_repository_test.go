package repository_test

import (
	"strings"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/popeskul/insdr-messenger/internal/models"
	"github.com/popeskul/insdr-messenger/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRepository_GetSentMessages_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewMessageRepository(db)

	tests := []struct {
		name           string
		setupData      func() error
		offset         int
		limit          int
		expectedCount  int
		validateResult func(t *testing.T, messages []*models.Message)
	}{
		{
			name: "Get sent messages with pagination - first page",
			setupData: func() error {
				now := time.Now()
				err := insertBulkTestMessages(db.DB, 5, "+1234567890", "Sent message", string(models.MessageStatusSent), &now, time.Hour)
				if err != nil {
					return err
				}
				err = insertBulkTestMessages(db.DB, 3, "+0987654321", "Pending message", string(models.MessageStatusPending), nil, 0)
				if err != nil {
					return err
				}
				return insertBulkTestMessages(db.DB, 2, "+1111111111", "Failed message", string(models.MessageStatusFailed), nil, 0)
			},
			offset:        0,
			limit:         3,
			expectedCount: 3,
			validateResult: func(t *testing.T, messages []*models.Message) {
				for _, msg := range messages {
					assert.Equal(t, models.MessageStatusSent, msg.Status)
					assert.True(t, msg.SentAt.Valid)
					assert.False(t, msg.SentAt.Time.IsZero())
				}

				for i := 1; i < len(messages); i++ {
					assert.True(t, messages[i-1].SentAt.Time.After(messages[i].SentAt.Time) ||
						messages[i-1].SentAt.Time.Equal(messages[i].SentAt.Time),
						"Messages should be ordered by sent_at DESC")
				}
			},
		},
		{
			name: "Get sent messages with pagination - second page",
			setupData: func() error {
				now := time.Now()
				return insertBulkTestMessages(db.DB, 10, "+2234567890", "Message number", string(models.MessageStatusSent), &now, time.Minute)
			},
			offset:        5,
			limit:         3,
			expectedCount: 3,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Len(t, messages, 3)
				for _, msg := range messages {
					assert.Equal(t, models.MessageStatusSent, msg.Status)
				}
			},
		},
		{
			name: "Get sent messages - empty result",
			setupData: func() error {
				_, err := insertTestMessage(db.DB, "+3334445555", "Pending", string(models.MessageStatusPending), nil)
				if err != nil {
					return err
				}
				_, err = insertTestMessage(db.DB, "+4445556666", "Failed", string(models.MessageStatusFailed), nil)
				return err
			},
			offset:        0,
			limit:         10,
			expectedCount: 0,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Empty(t, messages)
			},
		},
		{
			name: "Get sent messages with message_id and no error",
			setupData: func() error {
				now := time.Now()
				messageID := "msg_123456789"
				_, err := insertTestMessageWithDetails(db.DB,
					"+5556667777",
					"Successfully sent message",
					string(models.MessageStatusSent),
					&messageID,
					nil,
					&now)
				return err
			},
			offset:        0,
			limit:         10,
			expectedCount: 1,
			validateResult: func(t *testing.T, messages []*models.Message) {
				require.Len(t, messages, 1)
				msg := messages[0]
				assert.Equal(t, models.MessageStatusSent, msg.Status)
				assert.True(t, msg.MessageID.Valid)
				assert.Equal(t, "msg_123456789", msg.MessageID.String)
				assert.False(t, msg.Error.Valid)
			},
		},
		{
			name: "Get sent messages - verify all fields are properly loaded",
			setupData: func() error {
				sentAt := time.Now().Add(-1 * time.Hour)
				messageID := "webhook_msg_id_12345"
				_, err := insertTestMessageWithDetails(db.DB,
					"+9998887777",
					"Test message with all fields",
					string(models.MessageStatusSent),
					&messageID,
					nil,
					&sentAt)
				return err
			},
			offset:        0,
			limit:         1,
			expectedCount: 1,
			validateResult: func(t *testing.T, messages []*models.Message) {
				require.Len(t, messages, 1)
				msg := messages[0]

				assert.NotZero(t, msg.ID)
				assert.Equal(t, "+9998887777", msg.PhoneNumber)
				assert.Equal(t, "Test message with all fields", msg.Content)
				assert.Equal(t, models.MessageStatusSent, msg.Status)
				assert.True(t, msg.MessageID.Valid)
				assert.Equal(t, "webhook_msg_id_12345", msg.MessageID.String)
				assert.False(t, msg.Error.Valid)
				assert.True(t, msg.SentAt.Valid)
				assert.False(t, msg.CreatedAt.IsZero())
				assert.False(t, msg.UpdatedAt.IsZero())
			},
		},
		{
			name: "Get sent messages with large offset",
			setupData: func() error {
				now := time.Now()
				return insertBulkTestMessages(db.DB, 3, "+7778889999", "Message", string(models.MessageStatusSent), &now, time.Hour)
			},
			offset:        10,
			limit:         5,
			expectedCount: 0,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Empty(t, messages)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(db)

			err := tt.setupData()
			require.NoError(t, err)

			messages, err := repo.GetSentMessages(tt.offset, tt.limit)

			assert.NoError(t, err)
			assert.Len(t, messages, tt.expectedCount)
			tt.validateResult(t, messages)
		})
	}
}

func TestMessageRepository_GetSentMessages_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func() repository.MessageRepository
		offset        int
		limit         int
		expectedError string
	}{
		{
			name: "Database connection closed",
			setupRepo: func() repository.MessageRepository {
				db, cleanup := setupTestDB(t)
				cleanup()
				return repository.NewMessageRepository(db)
			},
			offset:        0,
			limit:         10,
			expectedError: "database is closed",
		},
		{
			name: "Invalid database query due to SQL injection attempt",
			setupRepo: func() repository.MessageRepository {
				db, _ := setupTestDB(t)
				return repository.NewMessageRepository(db)
			},
			offset:        -1,
			limit:         -1,
			expectedError: "OFFSET must not be negative",
		},
		{
			name: "Zero limit returns empty result",
			setupRepo: func() repository.MessageRepository {
				db, _ := setupTestDB(t)
				cleanupTestData(db)
				now := time.Now()
				_, err := insertTestMessage(db.DB, "+1234567890", "Test message", string(models.MessageStatusSent), &now)
				require.NoError(t, err)
				return repository.NewMessageRepository(db)
			},
			offset:        0,
			limit:         0,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			messages, err := repo.GetSentMessages(tt.offset, tt.limit)

			if tt.name == "Zero limit returns empty result" {
				assert.NoError(t, err)
				if messages != nil {
					assert.Empty(t, messages)
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, messages)

				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

func TestMessageRepository_GetUnsentMessages_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewMessageRepository(db)

	tests := []struct {
		name           string
		setupData      func() error
		limit          int
		expectedCount  int
		validateResult func(t *testing.T, messages []*models.Message)
	}{
		{
			name: "Get unsent messages with limit",
			setupData: func() error {
				err := insertBulkTestMessages(db.DB, 5, "+1234567890", "Pending message", string(models.MessageStatusPending), nil, 0)
				if err != nil {
					return err
				}

				sentAt := time.Now().Add(-1 * time.Hour)
				messages := generateTestMessages(3, string(models.MessageStatusSent), "+0987654321", "Sent message", true)
				for i := range messages {
					messages[i].SentAt = &sentAt
					messages[i].MessageID = ptr("msg_sent_" + string(rune('1'+i)))
				}
				err = insertTestMessagesWithDetails(db.DB, messages)
				if err != nil {
					return err
				}

				failedMessages := generateTestMessages(2, string(models.MessageStatusFailed), "+1111111111", "Failed message", false)
				for i := range failedMessages {
					failedMessages[i].Error = ptr("Failed to send: network error")
				}
				return insertTestMessagesWithDetails(db.DB, failedMessages)
			},
			limit:         3,
			expectedCount: 3,
			validateResult: func(t *testing.T, messages []*models.Message) {
				for _, msg := range messages {
					assert.Equal(t, models.MessageStatusPending, msg.Status)
					assert.False(t, msg.SentAt.Valid, "SentAt should be null for pending messages")
					assert.False(t, msg.MessageID.Valid, "MessageID should be null for pending messages")
					assert.False(t, msg.Error.Valid, "Error should be null for pending messages")
				}

				for i := 1; i < len(messages); i++ {
					assert.True(t, messages[i].CreatedAt.After(messages[i-1].CreatedAt) ||
						messages[i].CreatedAt.Equal(messages[i-1].CreatedAt),
						"Messages should be ordered by created_at ASC")
				}
			},
		},
		{
			name: "Get all unsent messages when limit exceeds available",
			setupData: func() error {
				return insertBulkTestMessages(db.DB, 3, "+2234567890", "Pending msg", string(models.MessageStatusPending), nil, 0)
			},
			limit:         10,
			expectedCount: 3,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Len(t, messages, 3)
				for _, msg := range messages {
					assert.Equal(t, models.MessageStatusPending, msg.Status)
				}
			},
		},
		{
			name: "Get unsent messages - empty result",
			setupData: func() error {
				sentAt := time.Now()
				messageID := "msg_123"
				_, err := insertTestMessageWithDetails(db.DB,
					"+3334445555",
					"Sent",
					string(models.MessageStatusSent),
					&messageID,
					nil,
					&sentAt)
				if err != nil {
					return err
				}

				errorMsg := "Error occurred"
				_, err = insertTestMessageWithDetails(db.DB,
					"+4445556666",
					"Failed",
					string(models.MessageStatusFailed),
					nil,
					&errorMsg,
					nil)
				return err
			},
			limit:         10,
			expectedCount: 0,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Empty(t, messages)
			},
		},
		{
			name: "Get unsent messages - verify all fields are properly loaded",
			setupData: func() error {
				_, err := insertTestMessage(db.DB,
					"+9998887777",
					"Test pending message with all fields",
					string(models.MessageStatusPending),
					nil)
				return err
			},
			limit:         1,
			expectedCount: 1,
			validateResult: func(t *testing.T, messages []*models.Message) {
				require.Len(t, messages, 1)
				msg := messages[0]

				assert.NotZero(t, msg.ID)
				assert.Equal(t, "+9998887777", msg.PhoneNumber)
				assert.Equal(t, "Test pending message with all fields", msg.Content)
				assert.Equal(t, models.MessageStatusPending, msg.Status)
				assert.False(t, msg.MessageID.Valid)
				assert.False(t, msg.Error.Valid)
				assert.False(t, msg.SentAt.Valid)
				assert.False(t, msg.CreatedAt.IsZero())
				assert.False(t, msg.UpdatedAt.IsZero())
			},
		},
		{
			name: "Get unsent messages with exact limit",
			setupData: func() error {
				return insertBulkTestMessages(db.DB, 5, "+5556667777", "Message", string(models.MessageStatusPending), nil, 0)
			},
			limit:         5,
			expectedCount: 5,
			validateResult: func(t *testing.T, messages []*models.Message) {
				assert.Len(t, messages, 5)
				for i := 1; i < len(messages); i++ {
					assert.True(t, messages[i].CreatedAt.After(messages[i-1].CreatedAt) ||
						messages[i].CreatedAt.Equal(messages[i-1].CreatedAt))
				}
			},
		},
		{
			name: "Get unsent messages - check oldest first ordering",
			setupData: func() error {
				now := time.Now()
				timestamps := []time.Duration{
					-5 * time.Hour,
					-1 * time.Hour,
					-3 * time.Hour,
					-30 * time.Minute,
					-2 * time.Hour,
				}

				for i, offset := range timestamps {
					createdAt := now.Add(offset)
					query := `
						INSERT INTO messages (phone_number, content, status, created_at, updated_at)
						VALUES ($1, $2, $3, $4, $5)
					`
					_, err := db.DB.Exec(query,
						"+7778889999"+string(rune('0'+i)),
						"Message created at "+createdAt.Format("15:04:05"),
						string(models.MessageStatusPending),
						createdAt,
						createdAt)
					if err != nil {
						return err
					}
				}
				return nil
			},
			limit:         3,
			expectedCount: 3,
			validateResult: func(t *testing.T, messages []*models.Message) {
				require.Len(t, messages, 3)

				assert.Contains(t, messages[0].Content, "Message created at")
				assert.Contains(t, messages[1].Content, "Message created at")
				assert.Contains(t, messages[2].Content, "Message created at")

				assert.True(t, messages[0].CreatedAt.Before(messages[1].CreatedAt))
				assert.True(t, messages[1].CreatedAt.Before(messages[2].CreatedAt))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(db)

			err := tt.setupData()
			require.NoError(t, err)

			messages, err := repo.GetUnsentMessages(tt.limit)

			assert.NoError(t, err)
			assert.Len(t, messages, tt.expectedCount)
			tt.validateResult(t, messages)
		})
	}
}

func TestMessageRepository_GetUnsentMessages_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func() repository.MessageRepository
		limit         int
		expectedError string
	}{
		{
			name: "Database connection closed",
			setupRepo: func() repository.MessageRepository {
				db, cleanup := setupTestDB(t)
				cleanup()
				return repository.NewMessageRepository(db)
			},
			limit:         10,
			expectedError: "database is closed",
		},
		{
			name: "Negative limit",
			setupRepo: func() repository.MessageRepository {
				db, _ := setupTestDB(t)
				return repository.NewMessageRepository(db)
			},
			limit:         -1,
			expectedError: "LIMIT must not be negative",
		},
		{
			name: "Zero limit returns empty result",
			setupRepo: func() repository.MessageRepository {
				db, _ := setupTestDB(t)
				cleanupTestData(db)
				_, err := insertTestMessage(db.DB, "+1234567890", "Test message", string(models.MessageStatusPending), nil)
				require.NoError(t, err)
				return repository.NewMessageRepository(db)
			},
			limit:         0,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			messages, err := repo.GetUnsentMessages(tt.limit)

			if tt.name == "Zero limit returns empty result" {
				assert.NoError(t, err)
				if messages != nil {
					assert.Empty(t, messages)
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, messages)

				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

func TestMessageRepository_UpdateMessageStatus_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewMessageRepository(db)

	tests := []struct {
		name           string
		setupData      func() (int64, error)
		status         models.MessageStatus
		messageID      *string
		errorMsg       *string
		validateResult func(t *testing.T, messageID int64)
	}{
		{
			name: "Update pending message to sent with message_id",
			setupData: func() (int64, error) {
				return insertTestMessage(db.DB, "+1234567890", "Test message", string(models.MessageStatusPending), nil)
			},
			status:    models.MessageStatusSent,
			messageID: ptr("webhook_msg_123"),
			errorMsg:  nil,
			validateResult: func(t *testing.T, messageID int64) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE id = $1", messageID)
				require.NoError(t, err)

				assert.Equal(t, models.MessageStatusSent, msg.Status)
				assert.True(t, msg.MessageID.Valid)
				assert.Equal(t, "webhook_msg_123", msg.MessageID.String)
				assert.False(t, msg.Error.Valid)
				assert.True(t, msg.SentAt.Valid)
				assert.False(t, msg.SentAt.Time.IsZero())
			},
		},
		{
			name: "Update pending message to failed with error",
			setupData: func() (int64, error) {
				return insertTestMessage(db.DB, "+0987654321", "Failed message", string(models.MessageStatusPending), nil)
			},
			status:    models.MessageStatusFailed,
			messageID: nil,
			errorMsg:  ptr("Network timeout"),
			validateResult: func(t *testing.T, messageID int64) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE id = $1", messageID)
				require.NoError(t, err)

				assert.Equal(t, models.MessageStatusFailed, msg.Status)
				assert.False(t, msg.MessageID.Valid)
				assert.True(t, msg.Error.Valid)
				assert.Equal(t, "Network timeout", msg.Error.String)
				assert.False(t, msg.SentAt.Valid)
			},
		},
		{
			name: "Update sent message to failed",
			setupData: func() (int64, error) {
				sentAt := time.Now()
				messageID := "original_msg_id"
				return insertTestMessageWithDetails(db.DB, "+1111111111", "Retry message", string(models.MessageStatusSent), &messageID, nil, &sentAt)
			},
			status:    models.MessageStatusFailed,
			messageID: nil,
			errorMsg:  ptr("Webhook error: 503"),
			validateResult: func(t *testing.T, messageID int64) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE id = $1", messageID)
				require.NoError(t, err)

				assert.Equal(t, models.MessageStatusFailed, msg.Status)
				assert.False(t, msg.MessageID.Valid)
				assert.True(t, msg.Error.Valid)
				assert.Equal(t, "Webhook error: 503", msg.Error.String)
				assert.False(t, msg.SentAt.Valid)
			},
		},
		{
			name: "Update message status without changing message_id or error",
			setupData: func() (int64, error) {
				return insertTestMessage(db.DB, "+2223334444", "Status change only", string(models.MessageStatusPending), nil)
			},
			status:    models.MessageStatusSent,
			messageID: nil,
			errorMsg:  nil,
			validateResult: func(t *testing.T, messageID int64) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE id = $1", messageID)
				require.NoError(t, err)

				assert.Equal(t, models.MessageStatusSent, msg.Status)
				assert.False(t, msg.MessageID.Valid)
				assert.False(t, msg.Error.Valid)
				assert.True(t, msg.SentAt.Valid)
			},
		},
		{
			name: "Update failed message to sent clears error",
			setupData: func() (int64, error) {
				errorMsg := "Previous error"
				return insertTestMessageWithDetails(db.DB, "+5556667777", "Retry success", string(models.MessageStatusFailed), nil, &errorMsg, nil)
			},
			status:    models.MessageStatusSent,
			messageID: ptr("new_msg_id"),
			errorMsg:  nil,
			validateResult: func(t *testing.T, messageID int64) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE id = $1", messageID)
				require.NoError(t, err)

				assert.Equal(t, models.MessageStatusSent, msg.Status)
				assert.True(t, msg.MessageID.Valid)
				assert.Equal(t, "new_msg_id", msg.MessageID.String)
				assert.False(t, msg.Error.Valid)
				assert.True(t, msg.SentAt.Valid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(db)

			messageID, err := tt.setupData()
			require.NoError(t, err)

			err = repo.UpdateMessageStatus(messageID, tt.status, tt.messageID, tt.errorMsg)
			assert.NoError(t, err)

			tt.validateResult(t, messageID)
		})
	}
}

func TestMessageRepository_UpdateMessageStatus_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func() (repository.MessageRepository, int64)
		status        models.MessageStatus
		messageID     *string
		errorMsg      *string
		expectedError string
	}{
		{
			name: "Update non-existent message",
			setupRepo: func() (repository.MessageRepository, int64) {
				db, _ := setupTestDB(t)
				return repository.NewMessageRepository(db), 99999
			},
			status:        models.MessageStatusSent,
			messageID:     ptr("msg_123"),
			errorMsg:      nil,
			expectedError: "",
		},
		{
			name: "Database connection closed",
			setupRepo: func() (repository.MessageRepository, int64) {
				db, cleanup := setupTestDB(t)
				messageID, _ := insertTestMessage(db.DB, "+1234567890", "Test", string(models.MessageStatusPending), nil)
				cleanup()
				return repository.NewMessageRepository(db), messageID
			},
			status:        models.MessageStatusSent,
			messageID:     nil,
			errorMsg:      nil,
			expectedError: "database is closed",
		},
		{
			name: "Invalid status value",
			setupRepo: func() (repository.MessageRepository, int64) {
				db, _ := setupTestDB(t)
				messageID, _ := insertTestMessage(db.DB, "+1234567890", "Test", string(models.MessageStatusPending), nil)
				return repository.NewMessageRepository(db), messageID
			},
			status:        "invalid_status",
			messageID:     nil,
			errorMsg:      nil,
			expectedError: "new row for relation \"messages\" violates check constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, messageID := tt.setupRepo()

			err := repo.UpdateMessageStatus(messageID, tt.status, tt.messageID, tt.errorMsg)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				if tt.name == "Update non-existent message" {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestMessageRepository_GetTotalSentCount_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewMessageRepository(db)

	tests := []struct {
		name          string
		setupData     func() error
		expectedCount int64
	}{
		{
			name: "Count with no sent messages",
			setupData: func() error {
				err := insertBulkTestMessages(db.DB, 3, "+1234567890", "Pending", string(models.MessageStatusPending), nil, 0)
				if err != nil {
					return err
				}
				return insertBulkTestMessages(db.DB, 2, "+0987654321", "Failed", string(models.MessageStatusFailed), nil, 0)
			},
			expectedCount: 0,
		},
		{
			name: "Count with only sent messages",
			setupData: func() error {
				now := time.Now()
				return insertBulkTestMessages(db.DB, 5, "+1234567890", "Sent", string(models.MessageStatusSent), &now, time.Hour)
			},
			expectedCount: 5,
		},
		{
			name: "Count with mixed statuses",
			setupData: func() error {
				now := time.Now()
				err := insertBulkTestMessages(db.DB, 10, "+1111111111", "Sent", string(models.MessageStatusSent), &now, time.Minute)
				if err != nil {
					return err
				}
				err = insertBulkTestMessages(db.DB, 5, "+2222222222", "Pending", string(models.MessageStatusPending), nil, 0)
				if err != nil {
					return err
				}
				return insertBulkTestMessages(db.DB, 3, "+3333333333", "Failed", string(models.MessageStatusFailed), nil, 0)
			},
			expectedCount: 10,
		},
		{
			name: "Count with large number of sent messages",
			setupData: func() error {
				now := time.Now()
				return insertBulkTestMessages(db.DB, 100, "+5556667777", "Bulk sent", string(models.MessageStatusSent), &now, time.Second)
			},
			expectedCount: 100,
		},
		{
			name:          "Count with empty table",
			setupData:     func() error { return nil },
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(db)

			err := tt.setupData()
			require.NoError(t, err)

			count, err := repo.GetTotalSentCount()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestMessageRepository_GetTotalSentCount_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func() repository.MessageRepository
		expectedError string
	}{
		{
			name: "Database connection closed",
			setupRepo: func() repository.MessageRepository {
				db, cleanup := setupTestDB(t)
				cleanup()
				return repository.NewMessageRepository(db)
			},
			expectedError: "database is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()

			count, err := repo.GetTotalSentCount()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Equal(t, int64(0), count)
		})
	}
}

func TestMessageRepository_CreateMessage_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewMessageRepository(db)

	tests := []struct {
		name        string
		phoneNumber string
		content     string
		validate    func(t *testing.T)
	}{
		{
			name:        "Create simple message",
			phoneNumber: "+1234567890",
			content:     "Hello, this is a test message",
			validate: func(t *testing.T) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE phone_number = $1", "+1234567890")
				require.NoError(t, err)

				assert.Equal(t, "+1234567890", msg.PhoneNumber)
				assert.Equal(t, "Hello, this is a test message", msg.Content)
				assert.Equal(t, models.MessageStatusPending, msg.Status)
				assert.False(t, msg.MessageID.Valid)
				assert.False(t, msg.Error.Valid)
				assert.False(t, msg.SentAt.Valid)
				assert.False(t, msg.CreatedAt.IsZero())
				assert.False(t, msg.UpdatedAt.IsZero())
			},
		},
		{
			name:        "Create message with special characters",
			phoneNumber: "+44-789-012-3456",
			content:     "Special chars: @#$%^&*()_+-=[]{}|;':\",./<>?",
			validate: func(t *testing.T) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE phone_number = $1", "+44-789-012-3456")
				require.NoError(t, err)

				assert.Equal(t, "+44-789-012-3456", msg.PhoneNumber)
				assert.Equal(t, "Special chars: @#$%^&*()_+-=[]{}|;':\",./<>?", msg.Content)
				assert.Equal(t, models.MessageStatusPending, msg.Status)
			},
		},
		{
			name:        "Create message with max length content",
			phoneNumber: "+9876543210",
			content:     strings.Repeat("A", 160),
			validate: func(t *testing.T) {
				var count int
				err := db.Get(&count, "SELECT COUNT(*) FROM messages WHERE phone_number = $1", "+9876543210")
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name:        "Create message with unicode content",
			phoneNumber: "+33123456789",
			content:     "Unicode test: 你好世界 🌍 Привет мир",
			validate: func(t *testing.T) {
				var msg models.Message
				err := db.Get(&msg, "SELECT * FROM messages WHERE phone_number = $1", "+33123456789")
				require.NoError(t, err)

				assert.Equal(t, "Unicode test: 你好世界 🌍 Привет мир", msg.Content)
			},
		},
		{
			name:        "Create multiple messages for same phone number",
			phoneNumber: "+1111111111",
			content:     "Message 1",
			validate: func(t *testing.T) {
				err := repo.CreateMessage("+1111111111", "Message 2")
				require.NoError(t, err)

				err = repo.CreateMessage("+1111111111", "Message 3")
				require.NoError(t, err)

				var count int
				err = db.Get(&count, "SELECT COUNT(*) FROM messages WHERE phone_number = $1", "+1111111111")
				require.NoError(t, err)
				assert.Equal(t, 3, count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(db)

			err := repo.CreateMessage(tt.phoneNumber, tt.content)
			assert.NoError(t, err)

			tt.validate(t)
		})
	}
}

func TestMessageRepository_CreateMessage_Failure(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name          string
		setupRepo     func() repository.MessageRepository
		phoneNumber   string
		content       string
		expectedError string
	}{
		{
			name: "Database connection closed",
			setupRepo: func() repository.MessageRepository {
				db, cleanup := setupTestDB(t)
				cleanup()
				return repository.NewMessageRepository(db)
			},
			phoneNumber:   "+1234567890",
			content:       "Test message",
			expectedError: "database is closed",
		},
		{
			name: "Content exceeds max length",
			setupRepo: func() repository.MessageRepository {
				return repository.NewMessageRepository(db)
			},
			phoneNumber:   "+1234567890",
			content:       strings.Repeat("B", 161),
			expectedError: "new row for relation \"messages\" violates check constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()

			err := repo.CreateMessage(tt.phoneNumber, tt.content)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
