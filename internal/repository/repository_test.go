package repository_test

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ppopeskul/insider-messenger/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryImpl_Message_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name     string
		validate func(t *testing.T, repo repository.Repository)
	}{
		{
			name: "Message repository is not nil",
			validate: func(t *testing.T, repo repository.Repository) {
				messageRepo := repo.Message()
				assert.NotNil(t, messageRepo)
			},
		},
		{
			name: "Message repository returns same instance",
			validate: func(t *testing.T, repo repository.Repository) {
				messageRepo1 := repo.Message()
				messageRepo2 := repo.Message()
				assert.Equal(t, messageRepo1, messageRepo2)
			},
		},
		{
			name: "Message repository methods are callable",
			validate: func(t *testing.T, repo repository.Repository) {
				messageRepo := repo.Message()
				
				_, err := messageRepo.GetUnsentMessages(10)
				assert.NoError(t, err)
				
				count, err := messageRepo.GetTotalSentCount()
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, count, int64(0))
				
				_, err = messageRepo.GetSentMessages(0, 10)
				assert.NoError(t, err)
			},
		},
		{
			name: "Message repository can create messages",
			validate: func(t *testing.T, repo repository.Repository) {
				messageRepo := repo.Message()
				
				err := messageRepo.CreateMessage("+1234567890", "Test message from repository test")
				assert.NoError(t, err)
				
				messages, err := messageRepo.GetUnsentMessages(10)
				assert.NoError(t, err)
				assert.Len(t, messages, 1)
				assert.Equal(t, "+1234567890", messages[0].PhoneNumber)
				assert.Equal(t, "Test message from repository test", messages[0].Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewRepository(db)
			tt.validate(t, repo)
			
			_, err := db.Exec("TRUNCATE TABLE messages RESTART IDENTITY CASCADE")
			require.NoError(t, err)
		})
	}
}

func TestRepositoryImpl_Message_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupDB       func() *sqlx.DB
	}{
		{
			name: "Repository with closed database connection",
			setupDB: func() *sqlx.DB {
				db, cleanup := setupTestDB(t)
				cleanup()
				return db
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB()
			repo := repository.NewRepository(db)
			messageRepo := repo.Message()
			assert.NotNil(t, messageRepo)
			
			_, err := messageRepo.GetUnsentMessages(10)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "database is closed")
		})
	}
}

func TestRepositoryImpl_Ping_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name     string
		setup    func()
		validate func(t *testing.T, err error)
	}{
		{
			name:  "Ping successful with healthy connection",
			setup: func() {},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "Multiple pings in succession",
			setup: func() {},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "Ping after database operations",
			setup: func() {
				_, err := db.Exec("SELECT 1")
				require.NoError(t, err)
			},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewRepository(db)
			
			tt.setup()
			
			err := repo.Ping()
			tt.validate(t, err)
			
			if tt.name == "Multiple pings in succession" {
				for i := 0; i < 5; i++ {
					err := repo.Ping()
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestRepositoryImpl_Ping_Failure(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func() repository.Repository
		expectedError string
		timeout       time.Duration
	}{
		{
			name: "Ping with closed database connection",
			setupRepo: func() repository.Repository {
				db, cleanup := setupTestDB(t)
				repo := repository.NewRepository(db)
				cleanup()
				return repo
			},
			expectedError: "database is closed",
			timeout:       3 * time.Second,
		},
		{
			name: "Ping with invalid connection string",
			setupRepo: func() repository.Repository {
				// Use a connection that will fail quickly
				db, err := sqlx.Open("postgres", "host=127.0.0.1 port=9999 user=test dbname=test sslmode=disable connect_timeout=1")
				require.NoError(t, err)
				return repository.NewRepository(db)
			},
			expectedError: "connection refused",
			timeout:       5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			
			done := make(chan bool)
			go func() {
				err := repo.Ping()
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				done <- true
			}()
			
			select {
			case <-done:
			case <-time.After(tt.timeout):
				t.Fatal("Ping timeout exceeded")
			}
		})
	}
}

func BenchmarkRepository_Ping(b *testing.B) {
	db, cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	
	repo := repository.NewRepository(db)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := repo.Ping()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_MessageAccess(b *testing.B) {
	db, cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	
	repo := repository.NewRepository(db)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		messageRepo := repo.Message()
		if messageRepo == nil {
			b.Fatal("Message repository is nil")
		}
	}
}
