APP_NAME=recommendation-service

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

seed:
	go run ./db/seed

tidy:
	go mod tidy