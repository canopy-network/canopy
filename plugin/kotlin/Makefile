# Makefile for Canopy Kotlin Plugin

.PHONY: all build clean test test-unit run dev lint format validate help

# Default target
all: build

# Build the project
build:
	@echo "Building Kotlin plugin..."
	./gradlew build

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	./gradlew clean
	rm -rf build/ out/

# Run the integration tests against a running Canopy node (matches Go's `make test`):
# delegates to the tutorial project, which runs the transaction tests AND the custom RPC endpoints
# test (/v1/query/faucets, /v1/query/rewards). Requires the node running with the kotlin plugin
# and its RPC server exposed on port 50010.
test:
	@echo "Running integration tests (transactions + custom RPC endpoints)..."
	cd tutorial && $(MAKE) test

# Unit tests for the main plugin (no node required)
test-unit:
	@echo "Running unit tests..."
	./gradlew test

# Run the application
run:
	@echo "Running plugin..."
	./gradlew run

# Run in development mode
dev:
	@echo "Running in development mode..."
	./gradlew dev

# Format code (manual formatting - ktlint removed due to protobuf conflicts)
format:
	@echo "Manual formatting required - ktlint disabled due to protobuf conflicts"

# Lint placeholder (ktlint removed)
lint:
	@echo "Linting disabled - ktlint has conflicts with generated protobuf files"

# Type check (compile without building jar)
type-check:
	@echo "Type checking..."
	./gradlew compileKotlin

# Run all validation checks (uses the node-less unit tests)
validate: lint type-check test-unit
	@echo "All validation checks passed!"

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	./gradlew generateProto

# Build JAR file
jar:
	@echo "Building JAR..."
	./gradlew jar

# Build fat JAR with all dependencies (same as root Makefile)
fatjar:
	@echo "Building fat JAR..."
	./gradlew fatJar --no-daemon

# Run with debugging enabled
debug:
	@echo "Running with debug on port 5005..."
	./gradlew run --debug-jvm

# Show help
help:
	@echo "Available targets:"
	@echo "  make build      - Build the project"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run the integration tests (transactions + custom RPC endpoints; requires running Canopy node)"
	@echo "  make test-unit  - Run the main plugin unit tests (no node required)"
	@echo "  make run        - Run the application"
	@echo "  make dev        - Run in development mode"
	@echo "  make lint       - Run linter"
	@echo "  make lint-fix   - Fix lint issues"
	@echo "  make format     - Format code"
	@echo "  make type-check - Type check code"
	@echo "  make validate   - Run all validation checks"
	@echo "  make proto      - Generate protobuf files"
	@echo "  make jar        - Build JAR file"
	@echo "  make fatjar     - Build fat JAR with dependencies"
	@echo "  make debug      - Run with debugging enabled"
	@echo "  make help       - Show this help message"