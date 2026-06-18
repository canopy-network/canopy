# Makefile for Canopy Kotlin Plugin Tutorial

.PHONY: all build clean test proto help

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

# Run the integration tests (requires a running Canopy node with the kotlin plugin).
# ./gradlew test runs ALL tests in the RpcTest class: the transaction tests
# (testPluginTransactions) AND the custom RPC endpoints test (testPluginCustomRPCEndpoints,
# /v1/query/faucets, /v1/query/rewards) — the latter needs the plugin's RPC server on port 50010.
test:
	@echo "Running tests (transactions + custom RPC endpoints)..."
	./gradlew test

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
	@echo "  make test     - Run the integration tests (transactions + custom RPC endpoints; requires running Canopy node)"
	@echo "  make proto    - Generate protobuf files"
	@echo "  make help     - Show this help message"
	@echo ""
	@echo "Prerequisites for test:"
	@echo "  1. Canopy node must be running with the Kotlin plugin enabled"
	@echo "  2. The plugin must have faucet and reward transaction types registered"
	@echo "  3. The plugin's RPC server must be reachable on port 50010 (for the custom RPC test)"
	@echo "  4. See ../TUTORIAL.md for full setup instructions"
