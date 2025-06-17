# Architecture Overview

## How It Works

```
Every 2 minutes:
1. Scheduler wakes up
2. Fetches 2 pending messages from PostgreSQL
3. Sends each message to webhook endpoint
4. Updates status to 'sent' or 'failed'
5. Caches successful message IDs in Redis
```

## System Components

```
┌─────────────────────────────────────────────────────┐
│                  HTTP API (:8080)                    │
│                                                      │
│  GET  /health          - System health check        │
│  GET  /messages/sent   - List sent messages         │
│  POST /scheduler/start - Start message sending      │
│  POST /scheduler/stop  - Stop message sending       │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                 Business Logic                       │
│                                                      │
│  • Scheduler (Go ticker, no cron)                   │
│  • Message Service (sending logic)                  │
│  • Webhook Service (HTTP client + Circuit Breaker)  │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│                  Data Storage                        │
│                                                      │
│  • PostgreSQL - Message storage                     │
│  • Redis - Message ID cache                         │
└─────────────────────────────────────────────────────┘
```

## Code Structure

```
internal/
├── handler/       # HTTP endpoints
├── service/       # Business logic
├── repository/    # Database operations
├── scheduler/     # Automatic sending
└── middleware/    # Request processing
```

## Key Implementation Details

### Message Processing
```go
// Fetch messages with lock to prevent duplicates
SELECT * FROM messages 
WHERE status = 'pending' 
ORDER BY created_at 
LIMIT 2 
FOR UPDATE SKIP LOCKED
```

### Circuit Breaker
Protects against webhook failures:
- **Closed**: Normal operation
- **Open**: Fails fast after too many errors
- **Half-Open**: Tests if service recovered

### Concurrent Processing
Messages are sent in parallel using goroutines with semaphore for rate limiting.

## Database Schema

```sql
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL CHECK (char_length(content) <= 160),
    status VARCHAR(20) DEFAULT 'pending',
    message_id VARCHAR(100),    -- External ID from webhook
    error TEXT,                  -- Error message if failed
    sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Message Flow Example

```
1. Add message to DB:
   INSERT INTO messages (phone_number, content) 
   VALUES ('+1234567890', 'Hello');

2. Scheduler picks it up:
   Status: pending → processing

3. Webhook call:
   POST https://webhook.site/xyz
   {"to": "+1234567890", "content": "Hello"}

4. Success response:
   {"messageId": "abc-123"}

5. Update DB:
   Status: processing → sent
   message_id: abc-123
   sent_at: 2025-01-17 10:30:00

6. Cache in Redis:
   SET message:abc-123 = timestamp (TTL 7 days)
```

## Configuration

Key settings in `config.docker.yaml`:
- `scheduler.interval_minutes`: How often to check (default: 2)
- `scheduler.batch_size`: Messages per batch (default: 2)  
- `webhook.url`: Where to send messages
- `webhook.timeout`: HTTP timeout in seconds