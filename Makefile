DB_URL ?= postgres://postgres:postgres@localhost:5432/centrachannel_db?sslmode=disable
MIGRATIONS_PATH ?= database/migrations
SEED_FILE ?= database/seed.sql
MIGRATE_BIN = $(shell go env GOPATH)/bin/migrate

define ensure_migrate
	@if [ ! -f "$(MIGRATE_BIN)" ]; then \
		echo "Installing golang-migrate..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
endef

.PHONY: migrate-up migrate-down migrate-drop seed reset status docker-build docker-run

migrate-up:
	$(call ensure_migrate)
	@echo "Running migrations up..."
	@$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up
	@echo "Migrations applied."

migrate-down:
	$(call ensure_migrate)
	@echo "Rolling back $(n) migration(s)..."
	@$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down $(or $(n),1)
	@echo "Rollback completed."

migrate-drop:
	$(call ensure_migrate)
	@echo "Dropping all tables..."
	@$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" drop -f
	@echo "All tables dropped."

seed:
	@echo "Running seeders..."
	@psql "$(DB_URL)" -f $(SEED_FILE)
	@echo "Seed completed."

reset: migrate-drop migrate-up seed
	@echo "Reset completed."

status:
	$(call ensure_migrate)
	@echo "Migration status:"
	@$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version 2>/dev/null || echo "No migrations applied yet."
	@echo ""
	@echo "Pending migrations:"
	@$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up --dry-run 2>/dev/null || echo "No pending migrations."

docker-build:
	@echo "Building Docker image for production..."
	@docker build -t centrachannel-app .
	@echo "Build completed."

docker-run:
	@echo "Running container for production..."
	@docker run -d --restart=always --env-file .env -p 3000:3000 --name centrachannel-app centrachannel-app
