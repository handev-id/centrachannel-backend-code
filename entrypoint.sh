#!/bin/sh
set -e

echo "Running migrations..."
migrate -path /root/database/migrations -database "$DB_URL" up

echo "Starting app..."
exec ./main
