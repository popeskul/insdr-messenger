// Package repository provides data access layer for the application.
package repository

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_repository.go -package=mocks github.com/ppopeskul/insdr-messenger/internal/repository Repository,MessageRepository
