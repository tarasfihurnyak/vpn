.PHONY: sqlc migrate-new migrate-up migrate-down test test-migration test-service

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
	go test -v -count=1 $(shell go list ./... | grep -v 'vpn/pkg/db')

test-migration:
	go test -v -count=1 ./pkg/db/...
