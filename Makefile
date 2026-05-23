.PHONY: run build tidy test sqlc docker-up docker-down migrate-up migrate-down

APP_NAME := inkspace-api
PKG := github.com/trishaneupnexx/inkspace-api
DB_URL ?= postgres://inkspace:inkspace@localhost:5432/inkspace?sslmode=disable

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/api

tidy:
	go mod tidy

test:
	go test ./...

sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc generate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
