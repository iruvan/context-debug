# Run the context-debug example without installing Postgres by hand.
#
#   make run      start Postgres (seeded) and the service
#   make request  curl the endpoint with debug mode on (in another shell)
#   make down     stop Postgres
#
# Override the compose command if you use the standalone binary:
#   make run COMPOSE=docker-compose

COMPOSE ?= docker compose
DSN     ?= postgres://demo:demo@localhost:5433/contextdebug?sslmode=disable
URL     ?= http://localhost:8023/test-1

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: up
up: ## Start Postgres only (creates the table and seeds it on first run)
	$(COMPOSE) up -d --wait postgres

.PHONY: run
run: up ## Start Postgres, then run the service against it
	@echo "listening on $(URL) — try: make request"
	POSTGRES_DSN="$(DSN)" go run .

.PHONY: request
request: ## Call the endpoint with debug mode enabled
	curl -sS -H "is-debug: 1" $(URL)

.PHONY: psql
psql: ## Open a psql shell on the example database
	$(COMPOSE) exec postgres psql -U demo -d contextdebug

.PHONY: down
down: ## Stop Postgres (keeps the data)
	$(COMPOSE) down

.PHONY: clean
clean: ## Stop Postgres and delete its data
	$(COMPOSE) down -v --remove-orphans
