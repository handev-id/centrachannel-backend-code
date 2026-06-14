#!/bin/bash
set -euo pipefail

MIGRATIONS_PATH="${MIGRATIONS_PATH:-database/migrations}"
DB_URL="${DB_URL:-postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable}"

echo "Migration status:"
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" version 2>/dev/null || echo "No migrations applied yet."
echo ""
echo "Pending migrations:"
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" up --dry-run 2>/dev/null || echo "No pending migrations."
