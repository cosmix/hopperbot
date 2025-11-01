# Hopperbot Makefile
# Build and development automation for the Hopperbot Slack bot

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Binary output
BINARY_NAME := hopperbot
BUILD_DIR := .
CMD_DIR := cmd/hopperbot

# Go build flags
LDFLAGS := -X main.version=$(VERSION) \
           -X main.commit=$(COMMIT) \
           -X main.buildTime=$(BUILD_TIME)

# Go tools
GOFMT := go fmt
GOVET := go vet
GOTEST := go test
GOBUILD := go build
GOMOD := go mod

.PHONY: all build test clean fmt vet tidy run dev help install-tools check coverage docker-build version

# Default target
all: clean fmt vet tidy test build

## help: Display this help message
help:
	@echo "Hopperbot - Available make targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          - Build the hopperbot binary with version info"
	@echo "  clean          - Remove built binaries"
	@echo "  install        - Build and install to GOPATH/bin"
	@echo ""
	@echo "Development targets:"
	@echo "  run            - Build and run the application with .env"
	@echo "  dev            - Run directly with 'go run' and .env (faster for development)"
	@echo "  fmt            - Format all Go source files"
	@echo "  vet            - Run go vet on all packages"
	@echo "  tidy           - Tidy and verify go.mod"
	@echo ""
	@echo "Testing targets:"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run all tests with verbose output"
	@echo "  coverage       - Run tests with coverage report"
	@echo "  coverage-html  - Generate HTML coverage report"
	@echo ""
	@echo "Quality targets:"
	@echo "  check          - Run fmt, vet, tidy, and test (pre-commit check)"
	@echo "  install-tools  - Install development tools"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo ""
	@echo "Utility targets:"
	@echo "  version        - Display build version information"
	@echo "  help           - Display this help message"
	@echo ""
	@echo "Current build info:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Go Version: $(GO_VERSION)"

## build: Build the hopperbot binary with embedded version information
build:
	@echo "Building hopperbot..."
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Build and install to GOPATH/bin
install:
	@echo "Installing hopperbot to $(GOPATH)/bin..."
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(GOPATH)/bin/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Installed: $(GOPATH)/bin/$(BINARY_NAME)"

## clean: Remove built binaries
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## run: Run the application locally (requires .env file)
run: build
	@echo "Starting hopperbot..."
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a && ./$(BINARY_NAME); \
	else \
		echo "Warning: .env file not found. Using environment variables."; \
		./$(BINARY_NAME); \
	fi

## dev: Run directly with 'go run' and .env (faster for development)
dev:
	@echo "Starting hopperbot in development mode..."
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a && go run $(CMD_DIR)/main.go; \
	else \
		echo "Warning: .env file not found. Using environment variables."; \
		go run $(CMD_DIR)/main.go; \
	fi

## fmt: Format all Go source files
fmt:
	@echo "Formatting Go files..."
	$(GOFMT) ./...
	@echo "Formatting complete"

## vet: Run go vet on all packages
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

## tidy: Tidy and verify go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy
	$(GOMOD) verify
	@echo "Tidy complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -timeout 30s ./...
	@echo "Tests complete"

## test-verbose: Run all tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -v -race -timeout 30s ./...

## coverage: Run tests with coverage report
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -timeout 30s -coverprofile=coverage.out -covermode=atomic ./...
	$(GOTEST) tool cover -func=coverage.out
	@echo ""
	@echo "Total coverage:"
	@$(GOTEST) tool cover -func=coverage.out | grep total | awk '{print $$3}'

## coverage-html: Generate HTML coverage report and open in browser
coverage-html:
	@echo "Generating HTML coverage report..."
	$(GOTEST) -race -timeout 30s -coverprofile=coverage.out -covermode=atomic ./...
	$(GOTEST) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Opening in browser..."
	@which open > /dev/null && open coverage.html || echo "Open coverage.html manually"

## check: Run all quality checks (pre-commit check)
check: fmt vet tidy test
	@echo ""
	@echo "✓ All checks passed!"

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t hopperbot:$(VERSION) \
		-t hopperbot:latest \
		.
	@echo "Docker image built: hopperbot:$(VERSION)"

## version: Display build version information
version:
	@echo "Hopperbot Build Information"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Go Version: $(GO_VERSION)"
