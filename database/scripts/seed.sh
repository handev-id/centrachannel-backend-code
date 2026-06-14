#!/bin/bash
set -euo pipefail

SEED_FILE="${SEED_FILE:-database/seed.sql}"
DB_URL="${DB_URL:-postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable}"

echo "Running seeders..."
psql "$DB_URL" -f "$SEED_FILE"
echo "Seed completed."
