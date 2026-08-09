.PHONY: up down logs test lint

up:
	docker compose -f docker-compose.yml up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f app

test:
	go test ./... -race -count=1 -cover

lint:
	golangci-lint run ./...
