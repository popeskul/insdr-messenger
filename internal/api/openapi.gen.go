// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oapi-codegen/runtime"
)

// Defines values for HealthResponseCircuitBreakerState.
const (
	Closed   HealthResponseCircuitBreakerState = "closed"
	HalfOpen HealthResponseCircuitBreakerState = "half-open"
	Open     HealthResponseCircuitBreakerState = "open"
)

// Defines values for HealthResponseDatabaseStatus.
const (
	HealthResponseDatabaseStatusConnected    HealthResponseDatabaseStatus = "connected"
	HealthResponseDatabaseStatusDisconnected HealthResponseDatabaseStatus = "disconnected"
)

// Defines values for HealthResponseRedisStatus.
const (
	HealthResponseRedisStatusConnected    HealthResponseRedisStatus = "connected"
	HealthResponseRedisStatusDisconnected HealthResponseRedisStatus = "disconnected"
)

// Defines values for HealthResponseSchedulerStatus.
const (
	HealthResponseSchedulerStatusRunning HealthResponseSchedulerStatus = "running"
	HealthResponseSchedulerStatusStopped HealthResponseSchedulerStatus = "stopped"
)

// Defines values for HealthResponseStatus.
const (
	Degraded  HealthResponseStatus = "degraded"
	Healthy   HealthResponseStatus = "healthy"
	Unhealthy HealthResponseStatus = "unhealthy"
)

// Defines values for MessageStatus.
const (
	Failed  MessageStatus = "failed"
	Pending MessageStatus = "pending"
	Sent    MessageStatus = "sent"
)

// Defines values for SchedulerResponseStatus.
const (
	SchedulerResponseStatusStarted SchedulerResponseStatus = "started"
	SchedulerResponseStatusStopped SchedulerResponseStatus = "stopped"
)

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse struct {
	// Error Error code
	Error string `json:"error"`

	// Message Human-readable error message
	Message string `json:"message"`

	// Timestamp Timestamp of the error
	Timestamp *time.Time `json:"timestamp"`
}

// HealthResponse defines model for HealthResponse.
type HealthResponse struct {
	// CircuitBreakerState Current circuit breaker state
	CircuitBreakerState *HealthResponseCircuitBreakerState `json:"circuit_breaker_state"`

	// CircuitBreakerStatus Circuit breaker statistics
	CircuitBreakerStatus *string `json:"circuit_breaker_status"`

	// DatabaseStatus Database connection status
	DatabaseStatus *HealthResponseDatabaseStatus `json:"database_status"`

	// RedisStatus Redis connection status
	RedisStatus *HealthResponseRedisStatus `json:"redis_status"`

	// SchedulerStatus Current scheduler status
	SchedulerStatus *HealthResponseSchedulerStatus `json:"scheduler_status"`

	// Status Service health status
	Status HealthResponseStatus `json:"status"`

	// Timestamp Current server timestamp
	Timestamp time.Time `json:"timestamp"`
}

// HealthResponseCircuitBreakerState Current circuit breaker state
type HealthResponseCircuitBreakerState string

// HealthResponseDatabaseStatus Database connection status
type HealthResponseDatabaseStatus string

// HealthResponseRedisStatus Redis connection status
type HealthResponseRedisStatus string

// HealthResponseSchedulerStatus Current scheduler status
type HealthResponseSchedulerStatus string

// HealthResponseStatus Service health status
type HealthResponseStatus string

// Message defines model for Message.
type Message struct {
	// Content Message content
	Content *string `json:"content,omitempty"`

	// Error Error message if sending failed
	Error *string `json:"error"`

	// Id Unique message identifier
	Id int64 `json:"id"`

	// MessageId External message ID from webhook response
	MessageId *string `json:"message_id"`

	// PhoneNumber Recipient phone number
	PhoneNumber string `json:"phone_number"`

	// SentAt Timestamp when the message was sent
	SentAt *time.Time `json:"sent_at,omitempty"`

	// Status Message sending status
	Status MessageStatus `json:"status"`
}

// MessageStatus Message sending status
type MessageStatus string

// MessageListResponse defines model for MessageListResponse.
type MessageListResponse struct {
	Messages   []Message  `json:"messages"`
	Pagination Pagination `json:"pagination"`
}

// Pagination defines model for Pagination.
type Pagination struct {
	// CurrentPage Current page number
	CurrentPage int `json:"current_page"`

	// ItemsPerPage Number of items per page
	ItemsPerPage int `json:"items_per_page"`

	// TotalItems Total number of items
	TotalItems int `json:"total_items"`

	// TotalPages Total number of pages
	TotalPages int `json:"total_pages"`
}

// SchedulerResponse defines model for SchedulerResponse.
type SchedulerResponse struct {
	// Message Status message
	Message string `json:"message"`

	// Status Current status of the scheduler
	Status SchedulerResponseStatus `json:"status"`
}

// SchedulerResponseStatus Current status of the scheduler
type SchedulerResponseStatus string

// GetSentMessagesParams defines parameters for GetSentMessages.
type GetSentMessagesParams struct {
	// Page Page number for pagination
	Page *int `form:"page,omitempty" json:"page,omitempty"`

	// Limit Number of items per page
	Limit *int `form:"limit,omitempty" json:"limit,omitempty"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Health check endpoint
	// (GET /health)
	HealthCheck(w http.ResponseWriter, r *http.Request)
	// Get list of sent messages
	// (GET /messages/sent)
	GetSentMessages(w http.ResponseWriter, r *http.Request, params GetSentMessagesParams)
	// Start automatic message sending
	// (POST /scheduler/start)
	StartScheduler(w http.ResponseWriter, r *http.Request)
	// Stop automatic message sending
	// (POST /scheduler/stop)
	StopScheduler(w http.ResponseWriter, r *http.Request)
}

// Unimplemented server implementation that returns http.StatusNotImplemented for each endpoint.

type Unimplemented struct{}

// Health check endpoint
// (GET /health)
func (_ Unimplemented) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get list of sent messages
// (GET /messages/sent)
func (_ Unimplemented) GetSentMessages(w http.ResponseWriter, r *http.Request, params GetSentMessagesParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Start automatic message sending
// (POST /scheduler/start)
func (_ Unimplemented) StartScheduler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Stop automatic message sending
// (POST /scheduler/stop)
func (_ Unimplemented) StopScheduler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// HealthCheck operation middleware
func (siw *ServerInterfaceWrapper) HealthCheck(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.HealthCheck(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetSentMessages operation middleware
func (siw *ServerInterfaceWrapper) GetSentMessages(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetSentMessagesParams

	// ------------- Optional query parameter "page" -------------

	err = runtime.BindQueryParameter("form", true, false, "page", r.URL.Query(), &params.Page)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "page", Err: err})
		return
	}

	// ------------- Optional query parameter "limit" -------------

	err = runtime.BindQueryParameter("form", true, false, "limit", r.URL.Query(), &params.Limit)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "limit", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetSentMessages(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// StartScheduler operation middleware
func (siw *ServerInterfaceWrapper) StartScheduler(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.StartScheduler(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// StopScheduler operation middleware
func (siw *ServerInterfaceWrapper) StopScheduler(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.StopScheduler(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL          string
	BaseRouter       chi.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/health", wrapper.HealthCheck)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/messages/sent", wrapper.GetSentMessages)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/scheduler/start", wrapper.StartScheduler)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/scheduler/stop", wrapper.StopScheduler)
	})

	return r
}
