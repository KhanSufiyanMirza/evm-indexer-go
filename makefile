.PHONY: migrate run

migrate:
	migrate -path db/migrations -database "postgresql://evm_indexer:strongpassword@localhost:5432/evm_indexer_go?sslmode=disable" up
run: 
	export $(cat .env.local | xargs) && go run cmd/server/main.go