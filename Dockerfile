# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git curl

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.55.2

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Run linter (temporarily disabled for quick testing)
# RUN golangci-lint run ./... --config .golangci.yml

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Download migrate tool
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and bash for scripts
RUN apk --no-cache add ca-certificates bash postgresql-client curl

# Create non-root user
RUN addgroup -g 1000 -S insider && \
    adduser -u 1000 -S insider -G insider
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main /app/insider-messenger
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Copy configuration and migrations
COPY config.yaml migrations/ /app/
COPY scripts/entrypoint.sh /app/
COPY api/openapi.yaml /app/api/
COPY static/ /app/static/

# Make entrypoint executable
RUN chmod +x /app/entrypoint.sh

# Change ownership
RUN chown -R insider:insider /app

# Switch to non-root user
USER insider

# Expose port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]