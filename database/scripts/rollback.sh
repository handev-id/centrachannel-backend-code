#!/bin/bash
set -euo pipefail

MIGRATIONS_PATH="${MIGRATIONS_PATH:-database/migrations}"
DB_URL="${DB_URL:-postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable}"
STEPS="${1:-1}"

echo "Rolling back $STEPS migration(s)..."
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" down "$STEPS"
echo "Rollback completed."
