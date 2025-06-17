#!/bin/bash
set -e

echo "Starting Insider Messenger..."

# Always check if migrations table exists and run migrations if needed
if [ -n "$DATABASE_URL" ] || [ -n "$DATABASE_HOST" ]; then
    # Build database URL if not provided
    if [ -z "$DATABASE_URL" ]; then
        DATABASE_URL="postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_DBNAME}?sslmode=disable"
    fi
    
    echo "Checking database migrations..."
    # Wait for database to be ready
    until PGPASSWORD=$DATABASE_PASSWORD psql -h "${DATABASE_HOST:-postgres}" -U "${DATABASE_USER:-insider}" -d "${DATABASE_DBNAME:-insider_db}" -c '\q' 2>/dev/null; do
        echo "Waiting for database..."
        sleep 2
    done
    
    # Run migrations
    echo "Running migrations..."
    migrate -path /app/migrations -database "$DATABASE_URL" up || {
        echo "Migration failed or already up to date"
    }
fi

# Start the application
exec /app/insider-messenger "$@"