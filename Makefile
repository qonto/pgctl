GIT_SHA=$(shell git rev-parse HEAD)
GIT_CLOSEST_TAG=$(shell git describe --always --abbrev=0 --tags)
DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

STATIC_LINKING=-linkmode external -extldflags '-static' -s -w
MUSL=$(shell if which apk > /dev/null; then echo "musl"; fi)

BINARY_TARGET_PATH=bin/pgctl
BUILD_INFO="-X metadata.Version=$(GIT_CLOSEST_TAG) -X metadata.SHA=$(GIT_SHA) -X metadata.Date=$(Date)"
BUILD_CMD=CGO_ENABLED=1 go build -ldflags "$(BUILD_INFO)"
BUILD_ALPINE_CMD=CGO_ENABLED=1 go build -tags musl -ldflags "$(STATIC_LINKING) $(BUILD_INFO)"

.PHONY: all
all: build

.PHONY: build
build:
	$(BUILD_CMD) -o $(BINARY_TARGET_PATH) ./cmd

.PHONY: build-alpine
build-alpine:
	$(BUILD_ALPINE_CMD) -o $(BINARY_TARGET_PATH) ./cmd

.PHONY: test
test:
	go test -tags="$(MUSL)" `go list ./... | grep -v test/integration` -cover -race -covermode=atomic -coverprofile=c.out -short

.PHONY: build-with-coverage
build-with-coverage:
	@echo "🔧 Building pgctl binary with coverage instrumentation..."
	@mkdir -p coverage
	CGO_ENABLED=1 go build -cover -ldflags "$(BUILD_INFO)" -o $(BINARY_TARGET_PATH) ./cmd

.PHONY: test-integration
test-integration: build-with-coverage
	@echo "🧪 Running integration tests with coverage collection..."
	@mkdir -p coverage
	GOCOVERDIR=./coverage go test -v ./test/integration/... -race
	@echo "📊 Processing coverage data..."
	@go tool covdata textfmt -i=./coverage -o=coverage/coverage-integrations-tests.out
	@echo "📈 Integration test coverage report:"
	@go tool cover -func=coverage/coverage-integrations-tests.out
	@echo "🌐 HTML coverage report generated at: coverage/coverage-integrations-tests.html"
	@go tool cover -html=coverage/coverage-integrations-tests.out -o=coverage/coverage-integrations-tests.html

.PHONY: test-integration-clean
test-integration-clean:
	@echo "🧹 Cleaning up coverage data..."
	@rm -rf coverage/

.PHONY: check-lint-version
check-lint-version:
	golangci-lint version --format short | awk -F. '{ if ($$2 >= 59) exit 0; else exit 1 }' || (echo "Version is too old, should be >= 1.59 please upgrade" && exit 1)

.PHONY: lint
lint: check-lint-version
	golangci-lint run --timeout 5m ./...

.PHONY: lint-fix
lint-fix: check-lint-version
	golangci-lint run --timeout 5m --fix ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: generate
generate:
	go generate ./...

.PHONY: local-postgres
local-postgres:
	docker-compose -f dev/docker-compose.yaml up -d db

.PHONY: local-connect
local-connect:
	psql -h localhost -p 5432 -U postgres -d postgres


.PHONY: local-postgres-2
local-postgres-2:
	docker-compose -f dev/docker-compose.yaml up -d db2

.PHONY: local-connect-2
local-connect-2:
	psql -h localhost -p 5433 -U postgres -d postgres
