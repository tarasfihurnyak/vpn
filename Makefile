.PHONY: sqlc migrate-new migrate-up migrate-down test test-migration test-service lint

sqlc:
	docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate

migrate-new:
	@[ "$(NAME)" ] || { echo "usage: make migrate-new NAME=<name>"; exit 1; }
	go run ./cmd/migrate new $(NAME)

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

test: test-migration
	go test -v -race -count=1 $(shell go list ./... | grep -v 'vpn/pkg/db')

test-migration:
	go test -v -race -count=1 ./pkg/db/...

lint:
	docker run --rm \
		-v $(PWD):/app \
		-w /app \
		golangci/golangci-lint:latest \
		golangci-lint run ./...
