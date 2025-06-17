package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// repositoryImpl is the concrete implementation of Repository interface.
type repositoryImpl struct {
	db      *sqlx.DB
	message MessageRepository
}

// NewRepository creates a new repository instance.
func NewRepository(db *sqlx.DB) Repository {
	return &repositoryImpl{
		db:      db,
		message: NewMessageRepository(db),
	}
}

// Message returns the message repository.
func (r *repositoryImpl) Message() MessageRepository {
	return r.message
}

// Ping checks if the database connection is healthy.
func (r *repositoryImpl) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return r.db.PingContext(ctx)
}
