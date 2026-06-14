#!/bin/bash
set -euo pipefail

MIGRATIONS_PATH="${MIGRATIONS_PATH:-database/migrations}"
DB_URL="${DB_URL:-postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable}"
SEED_FILE="${SEED_FILE:-database/seed.sql}"

echo "Dropping all tables..."
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" drop -f

echo "Running migrations up..."
migrate -path "$MIGRATIONS_PATH" -database "$DB_URL" up

echo "Running seeders..."
psql "$DB_URL" -f "$SEED_FILE"

echo "Reset completed."
