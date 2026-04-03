.PHONY: sqlc

sqlc:
	docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate
