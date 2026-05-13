# Makefile for Canopy Kotlin Plugin Tutorial

.PHONY: all build clean test test-rpc proto help

# Default target
all: build

# Build the project
build:
	@echo "Building tutorial project..."
	./gradlew build -x test

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	./gradlew clean
	rm -rf build/

# Run all tests
test:
	@echo "Running tests..."
	./gradlew test

# Run RPC integration test (requires running Canopy node with plugin)
test-rpc:
	@echo "Running RPC integration test..."
	@echo "Prerequisites: Canopy node must be running with the Kotlin plugin enabled"
	@echo "               The plugin must have faucet and reward transaction types registered"
	@echo ""
	./gradlew test --tests "com.canopy.tutorial.RpcTest" --rerun-tasks

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	./gradlew generateProto

# Show help
help:
	@echo "Canopy Kotlin Plugin Tutorial"
	@echo "=============================="
	@echo ""
	@echo "Available targets:"
	@echo "  make build    - Build the project"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make test     - Run all tests"
	@echo "  make test-rpc - Run RPC integration test (requires running Canopy node)"
	@echo "  make proto    - Generate protobuf files"
	@echo "  make help     - Show this help message"
	@echo ""
	@echo "Prerequisites for test-rpc:"
	@echo "  1. Canopy node must be running with the Kotlin plugin enabled"
	@echo "  2. The plugin must have faucet and reward transaction types registered"
	@echo "  3. See ../TUTORIAL.md for full setup instructions"
