.DEFAULT_GOAL := test

POSTGRES_USER ?= postgres
POSTGRES_PW ?= postgres
POSTGRES_HOST ?= 127.0.0.1
POSTGRES_DB ?= postgres
POSTGRES_PORT ?= 5433
POSTGRES_SSL_MODE ?= disable
POSTGRES_URL ?= 'postgres://$(POSTGRES_USER):$(POSTGRES_PW)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)'

.PHONY: postgres-docker-start
postgres-docker-start:
	@ echo "Starting PostgreSQL in Docker..."
	@ docker compose up -d --wait srp-postgres

.PHONY: postgres-docker-stop
postgres-docker-stop:
	@ echo "Stopping PostgreSQL in Docker..."
	@ docker compose stop srp-postgres

.PHONY: postgres-docker-rm
postgres-docker-rm: postgres-docker-stop
	@ echo "Removing PostgreSQL Docker container..."
	@ docker compose rm srp-postgres

.PHONY: migrate
migrate:
	@ migrate -path migrations/ -database $(POSTGRES_URL) up

.PHONY: rebuild-postgres-db
rebuild-postgres-db: postgres-docker-rm postgres-docker-start migrate

.PHONY: mocks
mocks: ## generate mocks for tests
	mockery

.PHONY: test
test:
	@ go test -v -coverprofile=coverprofile.out -covermode=count ./...