.PHONY: all build test test-race cover bench lint fmt clean deps examples ci help

# Packages to build and test (excludes superjsonic, internal, validators)
PACKAGES = . ./builders/ ./builders/transformers/ ./builders/jsonschema/ ./query/

# Default target
all: test

# Build library packages
build:
	@echo "Building..."
	@go build $(PACKAGES)

# Run tests
test:
	@echo "Running tests..."
	@go test $(PACKAGES) -count=1 -timeout 60s

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test $(PACKAGES) -count=1 -race -timeout 60s

# Run tests with coverage
cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic $(PACKAGES)
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
cover-html: cover
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem $(PACKAGES)

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run --skip-dirs=superjsonic --timeout=5m || \
		echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt $(PACKAGES)

# Build all examples
examples:
	@echo "Building examples..."
	@for d in examples/*/; do \
		echo "  $$d"; \
		go build ./$$d... || exit 1; \
	done
	@echo "All examples built successfully"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@for d in examples/*/; do \
		name=$$(basename $$d); \
		rm -f $$name; \
	done

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Simulate CI pipeline locally
ci: build test-race examples
	@echo "CI simulation passed"

# Help
help:
	@echo "Available targets:"
	@echo "  make test       - Run tests"
	@echo "  make test-race  - Run tests with race detector"
	@echo "  make cover      - Run tests with coverage"
	@echo "  make cover-html - Generate HTML coverage report"
	@echo "  make bench      - Run benchmarks"
	@echo "  make lint       - Run linter"
	@echo "  make fmt        - Format code"
	@echo "  make build      - Build library packages"
	@echo "  make examples   - Build all examples"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make deps       - Install dependencies"
	@echo "  make ci         - Simulate CI pipeline locally"
