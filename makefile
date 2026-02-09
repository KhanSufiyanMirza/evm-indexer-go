DB_URL=postgresql://evm_indexer:strongpassword@localhost:5432/evm_indexer_go?sslmode=disable
BINARY_NAME=evm-indexer-go

.PHONY: migrate migrate-up migrate-down migrate-create custom-migrate-create sqlc test build run up down logs services help

migrate: migrate-up

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

run:
	export $$(cat .env.local | xargs) && go run cmd/server/main.go

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

services: up

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  up              Start docker services"
	@echo "  down            Stop docker services"
	@echo "  logs            View docker service logs"
	@echo "  migrate-up      Run database migrations up"
	@echo "  migrate-down    Rollback the last database migration"
	@echo "  migrate-create  Create a new migration file (interactive)"
	@echo "  sqlc            Generate Go code from SQL"
	@echo "  test            Run tests"
	@echo "  build           Build the application binary"
	@echo "  run             Run the application locally"
	@echo "  help            Show this help message"