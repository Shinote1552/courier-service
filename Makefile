ifeq ($(OS),Windows_NT)
    BIN_EXT := .exe
else
    BIN_EXT :=
endif

BIN_DIR := bin
SERVICE_API_DIR := api/openapi

ALL_PKGS := ./...
COVERAGE_UNIT_DIR := coverage-unit
COVERAGE_INTEGRATION_DIR := coverage-integration
COVERAGE_TOTAL_DIR := coverage
COVERAGE_PKGS := \
    ./internal/handlers/... \
    ./internal/service/... \
    ./internal/repository/... \
    ./internal/gateway/... \
    ./internal/entities/... \
    ./internal/pkg/... \
    ./pkg/...

GOLANGCI_LINT_BIN := $(BIN_DIR)/golangci-lint$(BIN_EXT)
OAPI_CODEGEN_BIN := $(BIN_DIR)/oapi-codegen$(BIN_EXT)
COVERAGE_MERGE_BIN := $(BIN_DIR)/gocovmerge$(BIN_EXT)
GOFUMPT_BIN := $(BIN_DIR)/gofumpt$(BIN_EXT)
MOCKGEN_BIN := $(BIN_DIR)/mockgen$(BIN_EXT)
GOOSE_BIN := $(BIN_DIR)/goose$(BIN_EXT)
WIRE_BIN := $(BIN_DIR)/wire$(BIN_EXT)
PROTOC_GEN_GO_BIN := $(BIN_DIR)/protoc-gen-go$(BIN_EXT)
PROTOC_GEN_GO_GRPC_BIN := $(BIN_DIR)/protoc-gen-go-grpc$(BIN_EXT)
PROTOLINT_BIN := $(BIN_DIR)/protolint$(BIN_EXT)
HEY_BIN := $(BIN_DIR)/hey$(BIN_EXT)


API_SCHEMA := courier-api.yaml
CODEGEN_CONFIG := codegen-config.yaml

# proto
PROTO_DIR := api/proto
PROTO_OUT_DIR := internal/generated/proto
PROTO_FILES := $(PROTO_DIR)/clients/orders.proto
PROTO_GO_FILES := $(PROTO_OUT_DIR)/orders/orders.pb.go
PROTO_GRPC_FILES := $(PROTO_OUT_DIR)/orders/orders_grpc.pb.go

ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

GOOSE_DBSTRING = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)
MIGRATIONS_DIR = migrations
GOOSE_DRIVER = postgres

POSTGRES_CONTAINER_NAME = ${COMPOSE_PROJECT_NAME}-postgres

.PHONY: all dev run dev-env debug test deps tools clean codegen fmt \
        migrate-up migrate-down migrate-status postgres-up postgres-wait \
        postgres-stop postgres-ref postgres-clean postgres-info postgres-health \
        connect-db wire-gen golangci lint dev-run mock-gen \
        coverage coverage-unit coverage-integration coverage-html \
        proto proto-gen proto-lint proto-clean proto-tidy

dev-run: setup-dirs dev-env deps postgres-up postgres-wait postgres-health migrate-up migrate-status postgres-info run
	@echo "Development environment started without database reset and tools"
	@echo ''

dev-general: setup-dirs dev-env deps tools codegen wire-gen fmt lint
	@echo "Ready to make run"
	@echo ''

dev-env:
	@cp -f .env.example .env

all: setup-dirs deps tools codegen mock-gen fmt lint

run:
	@go run service/cmd/service
	@echo ''

setup-dirs:
	@mkdir -p $(BIN_DIR)
	@echo ''

postgres-ref: postgres-stop postgres-clean postgres-up postgres-wait postgres-health migrate-up migrate-status postgres-info

postgres-up:
	@docker compose --env-file .env up -d

postgres-stop:
	@docker compose --env-file .env stop

postgres-clean:
	@docker compose --env-file .env down -v --remove-orphans
	@echo "Removing unused volumes..."

postgres-health:
	@docker exec $(POSTGRES_CONTAINER_NAME) pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}
	@echo ''

postgres-wait:
	@echo "Waiting for PostgreSQL to start... 3 sec"
	@sleep 3
	@echo ''

postgres-info:
	@echo "PostgreSQL containers: "
	@docker ps | grep postgres
	@echo ''
	@echo 'Docker PostgreSQL volumes'
	@docker volume ls | grep pgdata
	@echo ''

connect-db:
	@docker exec -it $(POSTGRES_CONTAINER_NAME) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

deps:
	@go mod download
	@go mod tidy
	@go mod verify
	@echo ''

lint: golangci

lint-fix: golangci-fix

codegen: api-gen wire-gen mock-gen proto-gen
	@echo "All code generation completed"

TESTS_DIR := ./internal/handlers/... ./internal/service/... ./pkg/token_bucket/... ./internal/gateway/grpc/order/...
test: mock-gen
	@go test --race $(TESTS_DIR)

test-parallel: mock-gen
	@echo "Running tests with parallel execution..."
	@go test --race -parallel=8 $(TESTS_DIR)

test-flaky: mock-gen
	@echo "Running test-flaky with -count=100..."
	@go test -count=100 --race -parallel=8 $(TESTS_DIR)

test-cover: mock-gen
	@echo "Running unit tests with coverage..."
	@mkdir -p $(COVERAGE_UNIT_DIR)
	@go test -coverpkg=$(COVERAGE_PKGS) -coverprofile=$(COVERAGE_UNIT_DIR)/coverage.out -covermode=atomic $(TESTS_DIR)
	@echo ""

test-cover-html: test-coverQ
	@echo "Generating HTML coverage report..."`
	@go tool cover -html=$(COVERAGE_UNIT_DIR)/coverage.out -o $(COVERAGE_UNIT_DIR)/coverage.html
	@echo "HTML coverage report generated: $(COVERAGE_UNIT_DIR)/coverage.html"
	
test-integration-cover:
	@make -f Makefile.test test-integration-cover

test-integration-cover-html:
	@make -f Makefile.test test-integration-cover-html

coverage-integration:
	@echo "Running integration tests with coverage..."
	@make -f Makefile.test test-integration-cover

coverage: $(COVERAGE_MERGE_BIN) test-cover coverage-integration
	@echo "Merging coverage reports..."
	@mkdir -p $(COVERAGE_TOTAL_DIR)
	@$(COVERAGE_MERGE_BIN) $(COVERAGE_UNIT_DIR)/coverage.out $(COVERAGE_INTEGRATION_DIR)/coverage.out > $(COVERAGE_TOTAL_DIR)/coverage.out 2>/dev/null || true
	@echo ""
	@echo "=== TOTAL COVERAGE (Unit + Integration) ==="
	@go tool cover -func=$(COVERAGE_TOTAL_DIR)/coverage.out
	@echo "==========================================="

coverage-html: coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_TOTAL_DIR)/coverage.out -o $(COVERAGE_TOTAL_DIR)/coverage.html
	@echo "HTML coverage report generated: $(COVERAGE_TOTAL_DIR)/coverage.html"

tools: $(OAPI_CODEGEN_BIN) \
	$(GOFUMPT_BIN) \
	$(GOOSE_BIN) \
	$(WIRE_BIN) \
	$(GOLANGCI_LINT_BIN) \
	$(MOCKGEN_BIN) \
	$(COVERAGE_MERGE_BIN) \
	$(PROTOC_GEN_GO_BIN) \
	$(PROTOC_GEN_GO_GRPC_BIN) \
	$(PROTOLINT_BIN) \
	$(HEY_BIN)

api-gen: $(OAPI_CODEGEN_BIN)
	@echo "Generating code from API schema..."
	$(OAPI_CODEGEN_BIN) -config $(SERVICE_API_DIR)/$(CODEGEN_CONFIG) $(SERVICE_API_DIR)/$(API_SCHEMA)

fmt: $(GOFUMPT_BIN)
	@$(GOFUMPT_BIN) -l -w .

migrate-up: $(GOOSE_BIN)
	@echo "Applying migrations from $(MIGRATIONS_DIR)..."
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING="$(GOOSE_DBSTRING)" $(GOOSE_BIN) -dir $(MIGRATIONS_DIR) up
	@echo ''

migrate-down: $(GOOSE_BIN)
	@echo "Rolling back migrations..."
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING="$(GOOSE_DBSTRING)" $(GOOSE_BIN) -dir $(MIGRATIONS_DIR) down
	@echo ''

migrate-status: $(GOOSE_BIN)
	@echo "Migration status in $(MIGRATIONS_DIR):"
	@GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING="$(GOOSE_DBSTRING)" $(GOOSE_BIN) -dir $(MIGRATIONS_DIR) status
	@echo ''

wire-gen: $(WIRE_BIN)
	@echo "Generating Wire dependencies..."
	@$(WIRE_BIN) gen ./internal/app/wire.go

mock-gen: $(MOCKGEN_BIN)
	@echo "Generating mocks..."
	@go generate ./internal/service/courier/...
	@go generate ./internal/service/delivery/...
	@go generate ./internal/handlers/rest/ping_get/...
	@go generate ./internal/handlers/rest/courier_get/...
	@go generate ./internal/handlers/rest/courier_post/...
	@go generate ./internal/handlers/rest/courier_put/...
	@go generate ./internal/handlers/rest/couriers_get/...
	@go generate ./internal/handlers/rest/delivery_assign_post/...
	@go generate ./internal/handlers/rest/delivery_unassign_post/...
	@go generate ./internal/gateway/grpc/order/...
	@go generate ./pkg/token_bucket/... 
	@echo "Mocks generated successfully"

proto: proto-lint proto-gen
	@echo "Proto lint and generation completed"

proto-gen: $(PROTOC_GEN_GO_BIN) $(PROTOC_GEN_GO_GRPC_BIN)
	@echo "Generating protobuf files..."
	@mkdir -p $(PROTO_OUT_DIR)
	@"$(PROTOC_GEN_GO_BIN)" --version
	@"$(PROTOC_GEN_GO_GRPC_BIN)" --version
	@protoc --proto_path=$(PROTO_DIR) \
		--plugin=protoc-gen-go="$(PROTOC_GEN_GO_BIN)" \
		--plugin=protoc-gen-go-grpc="$(PROTOC_GEN_GO_GRPC_BIN)" \
		--go_out=$(PROTO_OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@echo "Generated: $(PROTO_GO_FILES)"
	@echo "Generated: $(PROTO_GRPC_FILES)"

proto-lint: $(PROTOLINT_BIN)
	@echo "Linting proto files..."
	@$(PROTOLINT_BIN) lint $(PROTO_FILES)

proto-tidy:
	@echo "Formatting proto files..."
	@$(PROTOLINT_BIN) lint --fix $(PROTO_FILES)

proto-clean:
	@echo "Cleaning generated protobuf files..."
	@rm -rf $(PROTO_GO_FILES) $(PROTO_GRPC_FILES)
	@echo "Removed: $(PROTO_GO_FILES)"
	@echo "Removed: $(PROTO_GRPC_FILES)"

golangci: $(GOLANGCI_LINT_BIN)
	@$(GOLANGCI_LINT_BIN) run

golangci-fix: $(GOLANGCI_LINT_BIN)
	@$(GOLANGCI_LINT_BIN) run --fix

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(COVERAGE_UNIT_DIR) $(COVERAGE_INTEGRATION_DIR) $(COVERAGE_TOTAL_DIR)
	@echo ''

$(GOLANGCI_LINT_BIN):
	@echo "building golangci-lint"
	@go build -o $(GOLANGCI_LINT_BIN) github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	@$(GOLANGCI_LINT_BIN) --version

$(WIRE_BIN):
	@echo "building wire"
	@go build -o $(WIRE_BIN) github.com/google/wire/cmd/wire
	@echo "wire built successfully"

$(GOOSE_BIN):
	@echo "building goose"
	@go build -o $(GOOSE_BIN) github.com/pressly/goose/v3/cmd/goose
	@$(GOOSE_BIN) --version

$(GOFUMPT_BIN):
	@echo "building gofumpt"
	@go build -o $(GOFUMPT_BIN) mvdan.cc/gofumpt
	@$(GOFUMPT_BIN) --version

$(OAPI_CODEGEN_BIN):
	@echo "building oapi-codegen"
	@go build -o $(OAPI_CODEGEN_BIN) github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	@$(OAPI_CODEGEN_BIN) --version

$(MOCKGEN_BIN):
	@echo "building mockgen"
	@go build -o $(MOCKGEN_BIN) go.uber.org/mock/mockgen
	@$(MOCKGEN_BIN) -version

$(COVERAGE_MERGE_BIN):
	@echo "building gocovmerge"
	@go build -o $(COVERAGE_MERGE_BIN) github.com/wadey/gocovmerge
	@echo "gocovmerge built successfully"

$(PROTOC_GEN_GO_BIN):
	@echo "building protoc-gen-go"
	@go build -o $(PROTOC_GEN_GO_BIN) google.golang.org/protobuf/cmd/protoc-gen-go
	@$(PROTOC_GEN_GO_BIN) --version

$(PROTOC_GEN_GO_GRPC_BIN):
	@echo "building protoc-gen-go-grpc"
	@go build -o $(PROTOC_GEN_GO_GRPC_BIN) google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@$(PROTOC_GEN_GO_GRPC_BIN) --version

$(PROTOLINT_BIN):
	@echo "building protolint"
	@go build -o $(PROTOLINT_BIN) github.com/yoheimuta/protolint/cmd/protolint
	@$(PROTOLINT_BIN) --version

$(HEY_BIN):
	@echo "building hey"
	@go build -o $(HEY_BIN) github.com/rakyll/hey
	@echo "hey built successfully (check version with: bin/hey -h)"