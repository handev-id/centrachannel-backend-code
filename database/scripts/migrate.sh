#!/bin/bash
set -euo pipefail

MIGRATIONS_PATH="${MIGRATIONS_PATH:-database/migrations}"
DB_URL="${DB_URL:-postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable}"

echo "Running migrations up..."
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" up
echo "Migrations applied successfully."
