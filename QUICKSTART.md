# Quick Start Guide for Beginners

## 🚀 5-Minute Setup

### Step 1: Install Docker
If you don't have Docker:
- **Mac/Windows**: Download [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- **Linux**: `curl -fsSL https://get.docker.com | sh`

### Step 2: Get the Code
```bash
git clone https://github.com/ppopeskul/insider-messenger.git
cd insider-messenger
```

### Step 3: Configure Webhook (IMPORTANT!)
1. Go to https://webhook.site
2. Copy your unique URL (looks like: https://webhook.site/abc123...)
3. Edit `config.docker.yaml` and replace the webhook URL:
```yaml
webhook:
  url: YOUR_WEBHOOK_URL_HERE  # <- Paste your URL here
```

### Step 4: Start Everything
```bash
docker-compose up -d
```

### Step 5: Add Test Messages
```bash
# Add 5 test messages
docker-compose exec postgres psql -U insider -d insider_db -c "
INSERT INTO messages (phone_number, content) VALUES 
('+1234567890', 'Hello World'),
('+1234567891', 'Test Message'),
('+1234567892', 'Insider Demo'),
('+1234567893', 'Automatic Send'),
('+1234567894', 'It Works!');"
```

## ✅ Verify It's Working

1. **Check Health** (should show "healthy"):
```bash
curl http://localhost:8080/health
```

2. **Wait 2 minutes** and check your webhook.site page - you should see 2 messages!

3. **See Sent Messages**:
```bash
curl http://localhost:8080/messages/sent
```

## 🎯 What You Just Built

- ⏰ **Scheduler**: Sends 2 messages every 2 minutes automatically
- 💾 **Database**: Stores all messages and their status
- 🚀 **API**: Control everything via REST endpoints
- 🛡️ **Resilience**: Circuit breaker protects against webhook failures
- 📊 **Monitoring**: Health checks and logs for debugging

## ❓ Common Questions

**Q: Messages not sending?**
- Check webhook.site is configured correctly
- Look at logs: `docker-compose logs app`

**Q: How to stop sending?**
```bash
curl -X POST http://localhost:8080/scheduler/stop
```

**Q: How to change sending interval?**
- Edit `config.docker.yaml` → `scheduler.interval_minutes`
- Restart: `docker-compose restart app`

## 📸 Visual Guide

### System Overview
```
Your Messages → Database → Scheduler (every 2 min) → Webhook → Success! ✅
                              ↓
                         (2 at a time)
```

### Message Flow
1. Add messages to database (status: "pending")
2. Scheduler picks 2 messages
3. Sends to webhook URL
4. Updates status to "sent"
5. Repeat every 2 minutes

## 🛠️ Next Steps

- View API docs: http://localhost:8080/swagger/
- Read full documentation: [README.md](README.md)
- Customize webhook auth key in config
- Scale by changing batch_size

---
**Need help?** Check logs: `docker-compose logs -f app`
