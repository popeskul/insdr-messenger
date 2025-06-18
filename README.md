# Insdr Messenger

Automatic message sending system that processes messages from PostgreSQL and sends them via webhook every 2 minutes.

> 🚀 **New to the project?** Check out our [Beginner's Quick Start Guide](QUICKSTART.md) for a 5-minute setup!

## Table of Contents
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
- [Development](#development)
- [Database Schema](#database-schema)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)

## Quick Start

### Prerequisites
- Docker 20.10+
- Docker Compose 1.29+
- Make

### 1. Clone and Run
```bash
# Clone repository
git clone https://github.com/ppopeskul/insdr-messenger.git
cd insdr-messenger

make dev
make db-seed

Webhook url: https://webhook.site/#!/view/6e479f4c-d00b-45fc-a0f6-22e8440a1861/
Swagger url: http://localhost:8080/swagger/
```

### 3. Monitor System
```bash
# Check system health
curl http://localhost:8080/health | jq

# View sent messages
curl http://localhost:8080/messages/sent | jq

# Watch real-time logs
docker-compose logs -f app
```

### What's Running?
- **API Server**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/
- **PostgreSQL**: localhost:5432 (user: insdr, db: insdr_db)
- **Redis**: localhost:6379
- **Scheduler**: Automatically sends 2 messages every 2 minutes

## Architecture

The system follows Clean Architecture principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     Insdr Messenger                        │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐     ┌───────────┐     ┌─────────────┐       │
│  │   HTTP   │────▶│ Business  │────▶│ Repository  │       │
│  │ Handlers │     │  Logic    │     │   Layer     │       │
│  └──────────┘     └───────────┘     └─────────────┘       │
│        │                │                    │               │
│        ▼                ▼                    ▼               │
│  ┌──────────┐     ┌───────────┐     ┌─────────────┐       │
│  │  Router  │     │ Scheduler │     │  Database   │       │
│  │   (Chi)  │     │  Service  │     │ (PostgreSQL)│       │
│  └──────────┘     └───────────┘     └─────────────┘       │
│                         │                    │               │
│                         ▼                    ▼               │
│                  ┌─────────────┐     ┌─────────────┐       │
│                  │   Webhook   │     │    Redis    │       │
│                  │   Service   │     │   (Cache)   │       │
│                  └─────────────┘     └─────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### Key Components:
- **Handler Layer**: HTTP request handling and validation
- **Service Layer**: Business logic and orchestration
- **Repository Layer**: Database operations and caching
- **Scheduler**: Automated message processing
- **Circuit Breaker**: Fault tolerance for webhook calls

For detailed architecture documentation, see [Architecture Overview](docs/architecture.md).

## API Endpoints

### Swagger UI
Access interactive API documentation at: http://localhost:8080/swagger/

### Health Check
```bash
GET /health
```

### Get Sent Messages
```bash
GET /messages/sent?page=1&limit=20
```

### Control Scheduler
```bash
POST /scheduler/start
POST /scheduler/stop
```

## Configuration

Edit `config.docker.yaml`:

```yaml
webhook:
  url: https://webhook.site/your-id
  auth_key: your-auth-key

scheduler:
  interval_minutes: 2
  batch_size: 2
```

## Database Schema

```sql
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL CHECK (char_length(content) <= 160),
    status VARCHAR(20) DEFAULT 'pending',
    message_id VARCHAR(100),
    error TEXT,
    sent_at TIMESTAMP WITH TIME ZONE
);
```

## Troubleshooting

### Messages not sending
```bash
# Check scheduler status
curl http://localhost:8080/health | jq .scheduler_status

# Check pending messages
docker-compose exec postgres psql -U insdr -d insdr_db \
-c "SELECT COUNT(*) FROM messages WHERE status='pending';"
```

### Circuit Breaker Open
```bash
# Check state
curl http://localhost:8080/health | jq .circuit_breaker_state

# Wait 60s for reset or restart
docker-compose restart app
```

### Port Already in Use
```bash
# Change port in docker-compose.yml
ports:
  - "8081:8080"  # Change 8080 to another port
```

## Development

### Generate Swagger UI
```bash
# Download and setup Swagger UI (first time only)
make swagger-setup

# Or simply
make swagger

# Clean Swagger UI files
make swagger-clean
```

### Commands
```bash
make help     # Show all commands
make up       # Start services
make down     # Stop services
make logs     # View logs
make seed     # Add test data
make clean    # Clear database
```

### Project Structure
```
├── cmd/server/      # Application entry
├── internal/
│   ├── handler/     # HTTP handlers
│   ├── service/     # Business logic
│   ├── repository/  # Database layer
│   └── scheduler/   # Task scheduler
├── migrations/      # Database migrations
└── api/            # OpenAPI spec
```

## Requirements Met

✅ Go-based scheduler (no cron)  
✅ 2 messages every 2 minutes  
✅ PostgreSQL for storage  
✅ Redis for caching message IDs  
✅ Start/Stop scheduler API  
✅ Get sent messages API  
✅ Circuit breaker for webhook  
✅ Clean architecture  
✅ Docker support  
✅ Swagger documentation
## API Documentation

### Interactive Documentation
Access Swagger UI at: http://localhost:8080/swagger/

### Endpoints

#### Health Check
```http
GET /health

Response: 200 OK
{
  "status": "healthy",
  "database_status": "connected",
  "redis_status": "connected",
  "scheduler_status": "running",
  "circuit_breaker_state": "closed",
  "timestamp": "2025-01-17T10:30:00Z"
}
```

#### Get Sent Messages
```http
GET /messages/sent?page=1&limit=20

Response: 200 OK
{
  "messages": [
    {
      "id": 1,
      "phone_number": "+905551234567",
      "content": "Your message text",
      "status": "sent",
      "message_id": "webhook-returned-id",
      "sent_at": "2025-01-17T10:25:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "items_per_page": 20,
    "total_items": 150,
    "total_pages": 8
  }
}
```

#### Scheduler Control
```http
POST /scheduler/start
Response: 200 OK
{"status": "started", "message": "Scheduler started successfully"}

POST /scheduler/stop  
Response: 200 OK
{"status": "stopped", "message": "Scheduler stopped successfully"}
```

### Webhook Format
The system sends messages to the configured webhook URL:

```http
POST https://webhook.site/your-unique-id
Content-Type: application/json
x-ins-auth-key: your-auth-key

{
  "to": "+905551234567",
  "content": "Message content here"
}

Expected Response:
200 OK or 202 Accepted
{
  "message": "Accepted",
  "messageId": "unique-message-id"
}
```

## Configuration

Configuration is managed via `config.docker.yaml`:

```yaml
# Server configuration
server:
  port: "8080"
  read_timeout: 10
  write_timeout: 10

# Database configuration
database:
  host: postgres
  port: 5432
  user: insdr
  password: password
  dbname: insdr_db
  sslmode: disable

# Redis configuration
redis:
  host: redis
  port: 6379
  password: ""
  db: 0

# Webhook configuration
webhook:
  url: https://webhook.site/your-unique-id
  auth_key: your-auth-key
  timeout: 30
  circuit_breaker:
    max_requests: 3      # Max requests in half-open state
    interval: 60         # Seconds to calculate failure rate
    timeout: 60          # Seconds to wait before retry
    failure_ratio: 0.6   # 60% failure rate triggers open
    consecutive_fails: 5 # Min requests before evaluation

# Scheduler configuration
scheduler:
  interval_minutes: 2    # Check interval
  batch_size: 2         # Messages per batch

# Middleware configuration
middleware:
  rate_limit: 100
  rate_limit_burst: 1000
  enable_cors: true
  allowed_origins:
    - "*"
```

## Development

### Linting

The project uses `golangci-lint v2.x` for code quality checks.

To install:
```bash
# macOS
brew install golangci-lint

# Linux/Windows
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

To run lint:
```bash
make lint
```

### Prerequisites for Local Development
- Go 1.23+
- PostgreSQL 15+
- Redis 7+
- Make

### Generate Swagger UI
```bash
# Download and setup Swagger UI assets
make swagger-setup

# Clean Swagger files
make swagger-clean
```

### Generate API Code from OpenAPI
```bash
# Install code generator and generate
make generate
```

### Useful Make Commands
```bash
make help         # Show all available commands
make up           # Start all services
make down         # Stop all services
make logs         # View application logs
make logs-all     # View all service logs
make restart      # Restart services
make rebuild      # Rebuild and restart
make seed         # Add test data to database
make db-shell     # Open PostgreSQL shell
make health       # Check system health
make clean        # Clean build artifacts
make test         # Run tests
make lint         # Run linter
```

### Project Structure
```
insdr-messenger/
├── cmd/server/         # Application entry point
├── internal/
│   ├── api/           # Generated OpenAPI code
│   ├── config/        # Configuration management
│   ├── handler/       # HTTP request handlers
│   ├── middleware/    # HTTP middleware
│   ├── models/        # Domain models
│   ├── repository/    # Database operations
│   ├── scheduler/     # Message scheduler
│   └── service/       # Business logic
├── migrations/        # Database migrations
├── api/              # OpenAPI specification
├── static/           # Static files (Swagger UI)
├── scripts/          # Helper scripts
└── docs/             # Documentation
```

## Database Schema

See [Architecture Documentation](docs/architecture.md#database-schema) for details.

## Troubleshooting

### Messages Not Sending

1. **Check scheduler status:**
   ```bash
   curl http://localhost:8080/health | jq .scheduler_status
   ```

2. **Verify pending messages exist:**
   ```bash
   docker-compose exec postgres psql -U insdr -d insdr_db \
   -c "SELECT COUNT(*) FROM messages WHERE status='pending';"
   ```

3. **Check application logs:**
   ```bash
   docker-compose logs app --tail 100 | grep ERROR
   ```

### Circuit Breaker Open

If webhook is failing repeatedly:
```bash
# Check circuit breaker state
curl http://localhost:8080/health | jq .circuit_breaker_state

# If "open", wait 60 seconds for automatic reset
# Or restart the application
docker-compose restart app
```

### Database Connection Issues
```bash
# Check PostgreSQL status
docker-compose ps postgres

# View PostgreSQL logs
docker-compose logs postgres --tail 50

# Test connection
docker-compose exec postgres pg_isready -U insdr
```

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Or change port in docker-compose.yml
ports:
  - "8081:8080"  # Change 8080 to another port
```

## Advanced Topics

### Manual Message Processing
```bash
# Stop automatic scheduler
curl -X POST http://localhost:8080/scheduler/stop

# Process specific messages manually
# Then restart scheduler
curl -X POST http://localhost:8080/scheduler/start
```

### Performance Tuning
- Increase `batch_size` in config for more messages per cycle
- Adjust `interval_minutes` for different processing frequency
- Scale horizontally by running multiple instances (with single scheduler)

### Monitoring Queries
```sql
-- Messages by status
SELECT status, COUNT(*) FROM messages GROUP BY status;

-- Average processing time
SELECT AVG(sent_at - created_at) as avg_processing_time 
FROM messages WHERE status = 'sent';

-- Failed messages in last hour
SELECT * FROM messages 
WHERE status = 'failed' 
AND updated_at > NOW() - INTERVAL '1 hour';
```

## Requirements Checklist

✅ Go-based scheduler without cron  
✅ Sends 2 messages every 2 minutes  
✅ PostgreSQL for message storage  
✅ Redis for caching message IDs  
✅ RESTful API with OpenAPI spec  
✅ Start/Stop scheduler endpoints  
✅ Get sent messages endpoint  
✅ Circuit breaker for webhook resilience  
✅ Clean architecture design  
✅ Docker and Docker Compose support  
✅ Comprehensive documentation  

## License

MIT License - see LICENSE file for details