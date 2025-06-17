package repository_test

import (
	"database/sql"
	"fmt"
	"time"
)

func insertTestMessage(db *sql.DB, phoneNumber, content, status string, sentAt *time.Time) (int64, error) {
	var id int64
	query := `
		INSERT INTO messages (phone_number, content, status, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	
	now := time.Now()
	err := db.QueryRow(query, phoneNumber, content, status, sentAt, now, now).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert test message: %w", err)
	}
	
	return id, nil
}

func insertTestMessageWithDetails(db *sql.DB, phoneNumber, content, status string, messageID, errorMsg *string, sentAt *time.Time) (int64, error) {
	var id int64
	query := `
		INSERT INTO messages (phone_number, content, status, message_id, error, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	
	now := time.Now()
	err := db.QueryRow(query, phoneNumber, content, status, messageID, errorMsg, sentAt, now, now).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert test message with details: %w", err)
	}
	
	return id, nil
}

func insertBulkTestMessages(db *sql.DB, count int, phonePrefix string, contentPrefix string, status string, baseTime *time.Time, timeIncrement time.Duration) error {
	for i := 0; i < count; i++ {
		phoneNumber := phonePrefix + string(rune('0'+i%10))
		content := contentPrefix + " " + string(rune('1'+i))
		
		var sentAt *time.Time
		if baseTime != nil {
			t := baseTime.Add(time.Duration(i) * timeIncrement)
			sentAt = &t
		}
		
		_, err := insertTestMessage(db, phoneNumber, content, status, sentAt)
		if err != nil {
			return fmt.Errorf("failed to insert message %d: %w", i, err)
		}
		
		if timeIncrement == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}
	return nil
}

func insertTestMessagesWithDetails(db *sql.DB, messages []struct {
	PhoneNumber string
	Content     string
	Status      string
	MessageID   *string
	Error       *string
	SentAt      *time.Time
}) error {
	for i, msg := range messages {
		_, err := insertTestMessageWithDetails(db, msg.PhoneNumber, msg.Content, msg.Status, msg.MessageID, msg.Error, msg.SentAt)
		if err != nil {
			return fmt.Errorf("failed to insert message %d: %w", i, err)
		}
	}
	return nil
}

func generateTestMessages(count int, status, phonePrefix, contentPrefix string, withSentAt bool) []struct {
	PhoneNumber string
	Content     string
	Status      string
	MessageID   *string
	Error       *string
	SentAt      *time.Time
} {
	messages := make([]struct {
		PhoneNumber string
		Content     string
		Status      string
		MessageID   *string
		Error       *string
		SentAt      *time.Time
	}, count)
	
	now := time.Now()
	for i := 0; i < count; i++ {
		messages[i].PhoneNumber = phonePrefix + string(rune('0'+i%10))
		messages[i].Content = contentPrefix + " " + string(rune('1'+i))
		messages[i].Status = status
		
		if withSentAt {
			sentAt := now.Add(time.Duration(i) * time.Hour)
			messages[i].SentAt = &sentAt
		}
		
		if status == "sent" {
			msgID := fmt.Sprintf("msg_%d", i)
			messages[i].MessageID = &msgID
		} else if status == "failed" {
			errMsg := "Failed to send: network error"
			messages[i].Error = &errMsg
		}
	}
	
	return messages
}

func ptr(s string) *string {
	return &s
}
