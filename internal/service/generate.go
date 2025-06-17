package service

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_services.go -package=mocks github.com/ppopeskul/insider-messenger/internal/service MessageService,SchedulerService,HealthService
