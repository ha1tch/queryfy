.PHONY: all build test bench lint clean help

# Default target
all: test

# Build the examples
build:
	@echo "Building examples..."
	@go build -o bin/basic ./examples/basic

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run example
run-example: build
	@echo "Running basic example..."
	@./bin/basic

# Help
help:
	@echo "Available targets:"
	@echo "  make test      - Run tests"
	@echo "  make cover     - Run tests with coverage"
	@echo "  make bench     - Run benchmarks"
	@echo "  make lint      - Run linter"
	@echo "  make fmt       - Format code"
	@echo "  make build     - Build examples"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make deps      - Install dependencies"
	@echo "  make run-example - Run basic example"
