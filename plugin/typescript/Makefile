# Makefile for Canopy TypeScript Plugin

.PHONY: all install build build-proto build-descriptors build-all clean test run dev lint format validate help

# Default target
all: build-all

# Install dependencies
install:
	@echo "Installing dependencies..."
	npm ci

# Build TypeScript only
build:
	@echo "Building TypeScript..."
	npm run build

# Build protobuf code
build-proto:
	@echo "Building protobuf..."
	npm run build:proto

# Build descriptors
build-descriptors:
	@echo "Building descriptors..."
	npm run build:descriptors

# Full build (same as root Makefile: npm ci && npm run build:all)
build-all: install
	@echo "Full build (proto + descriptors + TypeScript)..."
	npm run build:all

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf dist/
	rm -rf node_modules/

# Run tests (in tutorial directory)
test:
	@echo "Running tests..."
	cd tutorial && npm install && npm test

# Run the application
run:
	@echo "Running plugin..."
	npm start

# Run in development mode
dev:
	@echo "Running in development mode..."
	npm run dev

# Lint placeholder
lint:
	@echo "Linting not configured - add ESLint target if needed"

# Format placeholder
format:
	@echo "Formatting not configured - add Prettier target if needed"

# Full validation
validate: lint test
	@echo "All validation checks passed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  make install          - Install dependencies (npm ci)"
	@echo "  make build            - Build TypeScript only"
	@echo "  make build-proto      - Build protobuf code"
	@echo "  make build-descriptors - Build descriptors"
	@echo "  make build-all        - Full build (install + proto + descriptors + TypeScript)"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make test             - Run tests (tutorial)"
	@echo "  make run              - Run the application"
	@echo "  make dev              - Run in development mode"
	@echo "  make validate         - Run all validation checks"
	@echo "  make help             - Show this help message"
