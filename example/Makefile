# Run the context-debug example without installing Postgres by hand.
#
#   make run      start Postgres+NSQ (seeded) and the HTTP service
#   make run-bg   start Postgres+NSQ (seeded) and the NSQ consumer, in another shell
#   make request  curl the endpoint with debug mode on (in another shell)
#   make publish  publish a sample message to NSQ (in another shell)
#   make down     stop Postgres+NSQ
#
# Override the compose command if you use the standalone binary:
#   make run COMPOSE=docker-compose

COMPOSE ?= docker compose
DSN     ?= postgres://demo:demo@localhost:5433/contextdebug?sslmode=disable
URL     ?= http://localhost:8023/test-1
TOPIC   ?= test
MSG     ?= {"hello":"world"}

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: up
up: ## Start Postgres and NSQ (nsqlookupd, nsqd, nsqadmin)
	$(COMPOSE) up -d --wait postgres nsqlookupd nsqd nsqadmin

.PHONY: run
run: up ## Start Postgres + NSQ, then run the service against it
	@echo "listening on $(URL) — try: make request"
	@echo "nsqadmin at http://localhost:4171"
	POSTGRES_DSN="$(DSN)" go run .

.PHONY: run-bg
run-bg: up ## Start Postgres + NSQ, then run the consumer against it (run alongside `make run` in another shell)
	@echo "consumer running — Ctrl+C to stop"
	POSTGRES_DSN="$(DSN)" go run ./cmd/consumer

.PHONY: request
request: ## Call the endpoint with debug mode enabled
	curl -sS -H "is-debug: 1" $(URL)

.PHONY: publish
publish: ## Publish a sample message to NSQ (override with TOPIC=... MSG='{...}')
	curl -sS -X POST "http://localhost:4151/pub?topic=$(TOPIC)" -d '$(MSG)'

.PHONY: psql
psql: ## Open a psql shell on the example database
	$(COMPOSE) exec postgres psql -U demo -d contextdebug

.PHONY: down
down: ## Stop Postgres (keeps the data)
	$(COMPOSE) down

.PHONY: clean
clean: ## Stop Postgres and NSQ, and delete all their data
	$(COMPOSE) down -v --remove-orphans
