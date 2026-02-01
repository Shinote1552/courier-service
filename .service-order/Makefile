SHELL := /bin/bash

.PHONY: run test lint install generate down logs

# Run full stack via Docker Compose (Kafka, Kafka UI, service)
run:
	docker compose up -d --build

# Stop and remove containers
down:
	docker compose down -v

# Follow service logs
logs:
	docker compose logs -f service-order

test:
	go test -v ./... -race

lint:
	@gofumpt -l -w . || true
	@golangci-lint run || true

install:
	@echo "oapi-codegen install placeholder"

generate:
	@echo "codegen placeholder (oapi-codegen + easyjson)"
