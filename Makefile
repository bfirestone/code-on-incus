.PHONY: build install clean test test-coverage test-coverage-integration test-coverage-scenarios test-coverage-all test-unit test-integration test-scenarios test-all test-smoke test-e2e test-pexpect test-pexpect-setup lint fmt tidy help

# Binary name
BINARY_NAME=coi
BINARY_FULL=claude-on-incus

# Build directory
BUILD_DIR=.

# Installation directory
INSTALL_DIR=/usr/local/bin

# Coverage directory
COVERAGE_DIR=coverage

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/coi
	@ln -sf $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_FULL)

# Install to system
install: build
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo ln -sf $(INSTALL_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_FULL)

# Clean build artifacts
clean:
	@$(GOCLEAN)
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -f $(BUILD_DIR)/$(BINARY_FULL)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf dist
	@bash scripts/cleanup-pycache.sh

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -timeout 5m ./...

# Run smoke tests only (super fast, no Incus required)
test-smoke:
	@echo "Running smoke tests..."
	@$(GOTEST) -v -timeout 30s ./tests/e2e/ -run TestSmoke

# Run E2E tests (requires Incus)
test-e2e:
	@echo "Running E2E tests..."
	@$(GOTEST) -v -tags=integration -timeout 10m ./tests/e2e/

# Setup Python test dependencies
test-python-setup:
	@echo "Installing Python test dependencies..."
	@pip install -r tests/support/requirements.txt
	@pip install ruff

# Run Python integration tests (requires Incus)
test-python: build
	@echo "Running Python integration tests..."
	@if groups | grep -q incus-admin; then \
		pytest tests/ -v; \
	else \
		echo "Running with incus-admin group..."; \
		sg incus-admin -c "pytest tests/ -v"; \
	fi

# Run Python tests with output (for debugging)
test-python-debug: build
	@echo "Running Python tests with output..."
	@if groups | grep -q incus-admin; then \
		pytest tests/ -v -s; \
	else \
		echo "Running with incus-admin group..."; \
		sg incus-admin -c "pytest tests/ -v -s"; \
	fi

# Run only Python CLI tests (no Incus required)
test-python-cli:
	@echo "Running Python CLI tests..."
	@pytest tests/cli/ -v

# Lint Python tests
lint-python:
	@echo "Linting Python tests..."
	@ruff check tests/
	@ruff format --check tests/

# Run unit tests only (fast)
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short -race ./...

# Run tests with coverage (unit tests only)
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running unit tests with coverage..."
	@$(GOTEST) -v -short -race -coverprofile=$(COVERAGE_DIR)/coverage-unit.out -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage-unit.out -o $(COVERAGE_DIR)/coverage-unit.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-unit.out | grep total | awk '{print "Unit Test Coverage: " $$3}'
	@echo "Report: $(COVERAGE_DIR)/coverage-unit.html"

# Run integration tests with coverage (requires Incus)
test-coverage-integration:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running integration tests with coverage..."
	@$(GOTEST) -v -tags=integration -coverprofile=$(COVERAGE_DIR)/coverage-integration.out -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage-integration.out -o $(COVERAGE_DIR)/coverage-integration.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-integration.out | grep total | awk '{print "Integration Test Coverage: " $$3}'
	@echo "Report: $(COVERAGE_DIR)/coverage-integration.html"

# Run scenario tests with coverage (requires Incus)
test-coverage-scenarios:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running scenario tests with coverage..."
	@$(GOTEST) -v -tags="integration,scenarios" -timeout 30m -coverprofile=$(COVERAGE_DIR)/coverage-scenarios.out -covermode=atomic ./integrations/...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage-scenarios.out -o $(COVERAGE_DIR)/coverage-scenarios.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-scenarios.out | grep total | awk '{print "Scenario Test Coverage: " $$3}'
	@echo "Report: $(COVERAGE_DIR)/coverage-scenarios.html"

# Run all tests with coverage (unit + integration + scenarios)
test-coverage-all:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running all tests with coverage..."
	@$(GOTEST) -v -tags="integration,scenarios" -timeout 30m -coverprofile=$(COVERAGE_DIR)/coverage-all.out -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage-all.out -o $(COVERAGE_DIR)/coverage-all.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-all.out | grep total | awk '{print "Total Coverage: " $$3}'
	@echo "Report: $(COVERAGE_DIR)/coverage-all.html"

# Run integration tests (requires Incus)
test-integration:
	@$(GOTEST) -v -tags=integration ./...

# Run scenario tests (requires Incus, comprehensive)
test-scenarios:
	@echo "Running scenario tests..."
	@$(GOTEST) -v -tags="integration,scenarios" -timeout 30m ./integrations/...

# Run all tests (unit + integration + scenarios)
test-all:
	@echo "Running all tests..."
	@$(GOTEST) -v -tags="integration,scenarios" -timeout 30m ./...

# Tidy dependencies
tidy:
	@$(GOMOD) tidy

# Format code
fmt:
	@$(GOFMT) ./...

# Check formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not installed" && exit 1)
	@golangci-lint run --timeout 5m

# Run go vet
vet:
	@$(GOVET) ./...

# Check documentation coverage
doc-coverage:
	@bash scripts/doc-coverage.sh

# Run all checks (CI)
check: fmt-check vet lint test

# Run all checks including doc coverage
check-all: check doc-coverage

# Build for multiple platforms
build-all:
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/coi
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/coi
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/coi
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/coi

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for all platforms"
	@echo "  install       - Install to $(INSTALL_DIR)"
	@echo "  clean         - Remove build artifacts"
	@echo ""
	@echo "Testing (Go):"
	@echo "  test                      - Run all tests with race detector"
	@echo "  test-smoke                - Run smoke tests only (<30s, no Incus)"
	@echo "  test-unit                 - Run unit tests only (fast, no Incus)"
	@echo "  test-e2e                  - Run E2E binary tests (requires Incus)"
	@echo "  test-integration          - Run integration tests (requires Incus)"
	@echo "  test-scenarios            - Run scenario tests (requires Incus, comprehensive)"
	@echo "  test-all                  - Run all tests including scenarios"
	@echo ""
	@echo "Testing (Python):"
	@echo "  test-python-setup         - Install Python test dependencies"
	@echo "  test-python               - Run Python integration tests (requires Incus)"
	@echo "  test-python-debug         - Run Python tests with output (for debugging)"
	@echo "  test-python-cli           - Run Python CLI tests only (no Incus required)"
	@echo ""
	@echo "Coverage:"
	@echo "  test-coverage             - Unit tests with coverage report"
	@echo "  test-coverage-integration - Integration tests with coverage (requires Incus)"
	@echo "  test-coverage-scenarios   - Scenario tests with coverage (requires Incus)"
	@echo "  test-coverage-all         - All tests with coverage (requires Incus)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format Go code"
	@echo "  fmt-check     - Check Go code formatting"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint"
	@echo "  lint-python   - Lint and format check Python tests"
	@echo "  check         - Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Maintenance:"
	@echo "  tidy          - Tidy dependencies"
	@echo "  help          - Show this help"

# Default target
.DEFAULT_GOAL := build
