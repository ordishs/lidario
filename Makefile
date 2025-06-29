# Makefile for lidario - Go library for reading/writing LAS/LAZ files

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint
GOFMT=gofmt

# Build variables
BINARY_NAME=lidario
BUILD_DIR=build

# Platform detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    HOMEBREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo "/opt/homebrew")
    LASZIP_PREFIX := $(HOMEBREW_PREFIX)/opt/laszip
else ifeq ($(UNAME_S),Linux)
    LASZIP_PREFIX := /usr/local
endif

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build test lint fmt clean install-deps check-deps help

# Default target
all: check-deps lint test build

# Help target
help:
	@echo "Available targets:"
	@echo "  make all          - Run lint, test, and build"
	@echo "  make build        - Build the library"
	@echo "  make test         - Run all tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-laz     - Run only LAZ-specific tests"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make fmt          - Format code with gofmt"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install-deps - Install C dependencies (macOS/Linux)"
	@echo "  make check-deps   - Check if dependencies are installed"
	@echo "  make mod-tidy     - Update and clean go.mod dependencies"

# Check dependencies
check-deps:
	@echo "$(GREEN)Checking dependencies...$(NC)"
	@which $(GOCMD) > /dev/null || (echo "$(RED)Error: Go is not installed$(NC)" && exit 1)
	@echo "✓ Go installed: $(shell go version)"
	
	@if [ -d "$(LASZIP_PREFIX)" ]; then \
		echo "✓ LASzip found at: $(LASZIP_PREFIX)"; \
	else \
		echo "$(RED)✗ LASzip not found at: $(LASZIP_PREFIX)$(NC)"; \
		echo "  Run 'make install-deps' to install"; \
		exit 1; \
	fi
	
	@which $(GOLINT) > /dev/null 2>&1 || echo "$(YELLOW)Warning: golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"

# Install C dependencies
install-deps:
	@echo "$(GREEN)Installing C dependencies...$(NC)"
ifeq ($(UNAME_S),Darwin)
	@echo "Installing LASzip on macOS..."
	@which brew > /dev/null || (echo "$(RED)Error: Homebrew not installed. Visit https://brew.sh$(NC)" && exit 1)
	brew install laszip
else ifeq ($(UNAME_S),Linux)
	@echo "Installing LASzip on Linux..."
	@echo "$(YELLOW)Note: This requires sudo access$(NC)"
	# For Ubuntu/Debian
	@if which apt-get > /dev/null 2>&1; then \
		sudo apt-get update && sudo apt-get install -y liblaszip-dev; \
	elif which yum > /dev/null 2>&1; then \
		sudo yum install -y laszip-devel; \
	else \
		echo "$(RED)Unsupported Linux distribution. Please install LASzip manually.$(NC)"; \
		exit 1; \
	fi
else
	@echo "$(RED)Unsupported operating system: $(UNAME_S)$(NC)"
	@exit 1
endif
	@echo "$(GREEN)Dependencies installed successfully!$(NC)"

# Build the library
build: check-deps
	@echo "$(GREEN)Building lidario...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -v -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "$(GREEN)Build complete!$(NC)"

# Run tests
test: check-deps
	@echo "$(GREEN)Running tests...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -v -cover ./...

# Run tests with verbose output
test-verbose: check-deps
	@echo "$(GREEN)Running tests (verbose)...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -v -cover -count=1 ./...

# Run only LAZ-specific tests
test-laz: check-deps
	@echo "$(GREEN)Running LAZ tests...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -v -run TestLaz ./...

# Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@if which $(GOLINT) > /dev/null 2>&1; then \
		$(GOLINT) run --timeout=5m; \
	else \
		echo "$(YELLOW)golangci-lint not installed, running go vet instead...$(NC)"; \
		$(GOCMD) vet ./...; \
	fi

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) -w -s .
	@echo "$(GREEN)Code formatted!$(NC)"

# Update go.mod
mod-tidy:
	@echo "$(GREEN)Tidying go.mod...$(NC)"
	$(GOMOD) tidy
	@echo "$(GREEN)go.mod updated!$(NC)"

# Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f *.test
	@rm -f *.out
	@rm -rf testdata/*.las # Clean test output files
	@$(GOCMD) clean -cache
	@echo "$(GREEN)Clean complete!$(NC)"

# Development shortcuts
.PHONY: dev watch

# Development build (faster, no optimization)
dev: check-deps
	@echo "$(GREEN)Building (development mode)...$(NC)"
	CGO_ENABLED=1 $(GOBUILD) -gcflags="all=-N -l" .

# Watch for changes and rebuild (requires entr)
watch:
	@which entr > /dev/null || (echo "$(RED)Error: entr not installed. Install with: brew install entr$(NC)" && exit 1)
	@echo "$(GREEN)Watching for changes...$(NC)"
	find . -name "*.go" | entr -c make dev

# Benchmarks
.PHONY: bench bench-laz

bench: check-deps
	@echo "$(GREEN)Running benchmarks...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -bench=. -benchmem ./...

bench-laz: check-deps
	@echo "$(GREEN)Running LAZ benchmarks...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -bench=Laz -benchmem ./...

# Code coverage
.PHONY: coverage coverage-html

coverage: check-deps
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	CGO_ENABLED=1 $(GOTEST) -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -func=coverage.out

coverage-html: coverage
	@echo "$(GREEN)Generating HTML coverage report...$(NC)"
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# CI/CD targets
.PHONY: ci

ci: check-deps lint test
	@echo "$(GREEN)CI checks passed!$(NC)"

# Installation
.PHONY: install uninstall

install: build
	@echo "$(GREEN)Installing lidario...$(NC)"
	@$(GOCMD) install .
	@echo "$(GREEN)lidario installed!$(NC)"

uninstall:
	@echo "$(GREEN)Uninstalling lidario...$(NC)"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)lidario uninstalled!$(NC)"