SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DEFAULT_GOAL := help

ENV_FILE ?= .env
MIGRATE_VERSION ?= v4.19.0
MIGRATE_BIN ?= $(shell go env GOPATH)/bin/migrate

# Reuse the project's dotenv parser; .env is not a Makefile or shell script.
DOTENV := go run github.com/joho/godotenv/cmd/godotenv
shell_quote = '$(subst ','"'"',$(1))'

.PHONY: help check-env check-migrate migrate rollback migrate-up migrate-down migrate-drop seed reset status run docker-build docker-run test test-unit test-all coverage
.PHONY: _migrate-up _migrate-down _migrate-drop _seed _reset _status _run _docker-build _docker-run _test _test-all

help:
	@printf '%s\n' \
		'make migrate / migrate-up     Apply pending migrations' \
		'make rollback / migrate-down  Roll back migrations (n=1 by default)' \
		'make migrate-drop             Drop all database tables' \
		'make seed                     Run all database seeders' \
		'make reset                    Drop, migrate, then seed in order' \
		'make status                   Show current version and pending migration files' \
		'make run                      Run the backend' \
		'make docker-build             Build the configured Docker image' \
		'make docker-run               Run that image with the configured environment' \
		'make test / test-unit         All unit tests, no result cache, with coverage' \
		'make test-all                 Unit and integration tests, no result cache' \
		'make coverage                 Alias for make test (includes HTML report)' \
		'' \
		'Configuration: .env (override with ENV_FILE=path/to/file)' \
		'Database: DB_URL, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/DB_SSL_MODE'

check-env:
	@test -f $(call shell_quote,$(ENV_FILE)) || { echo 'Environment file missing. Copy .env.example to .env or set ENV_FILE.' >&2; exit 1; }

# Existing environment and make command-line overrides keep precedence, matching
# config.Load(). The implementation target receives the loaded .env values.
migrate-up migrate-down migrate-drop seed reset status run docker-build docker-run test test-all: check-env
	@$(DOTENV) -f $(call shell_quote,$(ENV_FILE)) $(MAKE) --no-print-directory _$@ ENV_FILE=$(call shell_quote,$(ENV_FILE))

migrate: migrate-up

rollback: migrate-down

test-unit coverage: test

check-migrate:
	@if [[ ! -x $(call shell_quote,$(MIGRATE_BIN)) ]]; then
		echo 'Installing golang-migrate $(MIGRATE_VERSION)...'
		GOBIN=$(call shell_quote,$(dir $(MIGRATE_BIN))) go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
	fi
	test -x $(call shell_quote,$(MIGRATE_BIN))

# Build the fallback URL at runtime and percent-encode its components. Passwords
# stay in shell variables instead of being expanded into recipe source by Make.
define database_config
urlencode() {
	local LC_ALL=C value="$$1" encoded='' char i
	for ((i=0; i<$${#value}; i++)); do
		char="$${value:i:1}"
		case "$$char" in
			[a-zA-Z0-9.~_-]) encoded+="$$char" ;;
			*) printf -v encoded '%s%%%02X' "$$encoded" "'$$char" ;;
		esac
	done
	printf '%s' "$$encoded"
}
db_url="$${DB_URL:-}"
if [[ -z "$$db_url" ]]; then
	db_host="$${DB_HOST:-localhost}"
	if [[ "$$db_host" == *:* && "$$db_host" != \[*\] ]]; then db_host="[$$db_host]"; fi
	db_url="postgres://$$(urlencode "$${DB_USER:-postgres}"):$$(urlencode "$${DB_PASSWORD-postgres}")@$$db_host:$${DB_PORT:-5432}/$$(urlencode "$${DB_NAME:-centrachannel_db}")?sslmode=$$(urlencode "$${DB_SSL_MODE:-disable}")"
fi
migrations_path="$${MIGRATIONS_PATH:-database/migrations}"
endef

_migrate-up _migrate-down _migrate-drop _status: check-migrate

_migrate-up:
	@$(database_config)
	echo 'Running migrations up...'
	$(call shell_quote,$(MIGRATE_BIN)) -path "$$migrations_path" -database "$$db_url" up
	echo 'Migrations applied.'

_migrate-down:
	@$(database_config)
	steps="$${n:-$${STEPS:-1}}"
	[[ "$$steps" =~ ^[1-9][0-9]*$$ ]] || { echo 'n must be a positive integer.' >&2; exit 1; }
	printf 'Rolling back %s migration(s)...\n' "$$steps"
	$(call shell_quote,$(MIGRATE_BIN)) -path "$$migrations_path" -database "$$db_url" down "$$steps"
	echo 'Rollback completed.'

_migrate-drop:
	@$(database_config)
	echo 'Dropping all tables...'
	$(call shell_quote,$(MIGRATE_BIN)) -path "$$migrations_path" -database "$$db_url" drop -f
	echo 'All tables dropped.'

_seed:
	@$(database_config)
	echo 'Running seeders...'
	psql -X --set=ON_ERROR_STOP=1 --single-transaction --dbname="$$db_url" --file="$${SEED_FILE:-database/seed.sql}"
	echo 'Seed completed.'

# Recursive calls preserve the order even when the caller uses make -j.
_reset:
	@$(MAKE) --no-print-directory _migrate-drop
	$(MAKE) --no-print-directory _migrate-up
	$(MAKE) --no-print-directory _seed
	echo 'Reset completed.'

_status:
	@$(database_config)
	echo 'Migration status:'
	if output="$$( $(call shell_quote,$(MIGRATE_BIN)) -path "$$migrations_path" -database "$$db_url" version 2>&1 )"; then
		printf '%s\n' "$$output"
		version="$${output%% *}"
		[[ "$$version" =~ ^[0-9]+$$ ]] || { echo 'Unexpected migration version output.' >&2; exit 1; }
	elif [[ "$$output" == 'error: no migration' ]]; then
		echo 'No migrations applied yet.'
		version=0
	else
		printf '%s\n' "$$output" >&2
		exit 1
	fi
	[[ -d "$$migrations_path" ]] || { echo 'Migrations directory not found.' >&2; exit 1; }
	printf '\nPending migration files:\n'
	pending=0
	shopt -s nullglob
	for file in "$$migrations_path"/*.up.sql; do
		name="$${file##*/}"
		file_version="$${name%%_*}"
		[[ "$$file_version" =~ ^[0-9]+$$ ]] || continue
		if ((10#$$file_version > 10#$$version)); then
			printf '%s\n' "$$name"
			pending=1
		fi
	done
	if ((pending == 0)); then echo 'No pending migration files.'; fi

_run:
	@go run ./cmd/server/main.go

_docker-build:
	@docker build -t "$${IMAGE_NAME:-centrachannel-app}:$${IMAGE_TAG:-local}" .

# Host networking matches docker-compose.yml and allows DB_HOST=127.0.0.1.
# The backend reads HOST and PORT directly from the configured env file.
_docker-run:
	@docker run -d --restart=always --network host --env-file $(call shell_quote,$(ENV_FILE)) --name "$${CONTAINER_NAME:-centrachannel-app}" "$${IMAGE_NAME:-centrachannel-app}:$${IMAGE_TAG:-local}"

# Generate coverage even when tests fail, but preserve the failing exit status.
# Remove old reports first so a build failure never presents stale coverage.
define run_tests
coverage_dir="$${COVERAGE_DIR:-coverage}"
profile="$$coverage_dir/$(1).out"
html="$$coverage_dir/$(1).html"
mkdir -p "$$coverage_dir"
rm -f -- "$$profile" "$$html"
cover_packages="$$(IFS=,; printf '%s' "$${packages[*]}")"
test_status=0
go test -count=1 -v -timeout="$${TEST_TIMEOUT:-5m}" -covermode=atomic -coverpkg="$$cover_packages" -coverprofile="$$profile" "$${packages[@]}" || test_status=$$?
if [[ -f "$$profile" ]]; then
	printf '\nCoverage per function:\n'
	go tool cover -func="$$profile"
	go tool cover -html="$$profile" -o "$$html"
	printf '\nHTML coverage report: %s\n' "$$html"
fi
if ((test_status != 0)); then
	echo 'Tests failed; coverage may be incomplete.' >&2
fi
exit "$$test_status"
endef

_test:
	@package_list="$$(go list ./...)"
	packages=()
	while IFS= read -r package; do
		case "$$package" in */test/integration|*/test/integration/*) continue ;; esac
		packages+=("$$package")
	done <<< "$$package_list"
	$(call run_tests,unit)

_test-all:
	@package_list="$$(go list ./...)"
	mapfile -t packages <<< "$$package_list"
	$(call run_tests,all)
