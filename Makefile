# todoer Makefile
# Provides convenient commands for building, testing, and running the todoer application

.PHONY: help build test test-core test-integration test-quick test-cli clean install

# Default target
help:
	@echo "📝 todoer - Daily Journal Task Manager"
	@echo "======================================"
	@echo ""
	@echo "Available commands:"
	@echo "  build           - Build the todoer binary"
	@echo "  install         - Install todoer to your PATH"
	@echo "  test            - Run all tests"
	@echo "  test-core       - Run core package tests only"
	@echo "  test-integration- Run integration tests only"
	@echo "  test-quick      - Run quick smoke tests"
	@echo "  test-cli        - Run full CLI integration tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build      # Build the application"
	@echo "  make test-quick # Run quick tests before committing"
	@echo "  make test       # Run full test suite"

# Build the application
build:
	@echo "🔨 Building todoer..."
	go build -o todoer ./cmd/todoer
	@echo "✅ Build complete: ./todoer"

# Install to PATH
install:
	@echo "📦 Installing todoer..."
	go install ./cmd/todoer
	@echo "✅ todoer installed to $(shell go env GOPATH)/bin/todoer"

# Run all tests (excluding potentially long-running CLI tests)
test:
	@echo "🧪 Running all tests..."
	@./run_tests.sh all

# Run core package tests
test-core:
	@echo "🧪 Running core package tests..."
	@./run_tests.sh core

# Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	@./run_tests.sh integration

# Run quick smoke tests
test-quick:
	@echo "🧪 Running quick tests..."
	@./run_tests.sh quick

# Run full CLI integration tests
test-cli:
	@echo "🧪 Running CLI integration tests..."
	@./run_tests.sh cli

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f todoer
	go clean
	@echo "✅ Clean complete"

# Development workflow targets
.PHONY: dev-test dev-build

# Quick development test (before committing)
dev-test: test-quick
	@echo "✅ Development tests passed!"

# Development build and test
dev-build: build test-quick
	@echo "✅ Build and tests completed successfully!"
