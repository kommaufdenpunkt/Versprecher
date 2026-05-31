.PHONY: build run test tidy migrate devdb

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

# DDL als Superuser (Konvention §3). Lokal: sudo -u postgres make migrate
migrate:
	scripts/migrate.sh

devdb:
	scripts/dev_db.sh
