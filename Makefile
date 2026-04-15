.PHONY: sqlc migrate-new migrate-up migrate-down test test-migration test-service lint tls-setup up

tls-setup:
	@command -v mkcert >/dev/null 2>&1 || { \
		echo "mkcert not found."; \
		echo "Install it:"; \
		echo "  macOS: brew install mkcert"; \
		echo "  Linux: https://github.com/FiloSottile/mkcert/releases"; \
		exit 1; \
	}
	@mkdir -p certs
	@if [ -f certs/cert.pem ]; then \
		echo "TLS certificates already exist"; \
	else \
		echo "Generating TLS certificates..."; \
		mkcert -install; \
		mkcert -key-file certs/key.pem -cert-file certs/cert.pem localhost 127.0.0.1 ::1; \
		echo "Certificates generated in certs/"; \
	fi

up: tls-setup
	docker compose up --build -d

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
