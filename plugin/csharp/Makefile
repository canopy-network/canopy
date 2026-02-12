.PHONY: restore build test test-tutorial lint format clean proto run serve serve-dev validate docs

# Development setup
restore:
	dotnet restore

# Build
build:
	dotnet build

build-release:
	dotnet build --configuration Release

# Testing
test:
	dotnet test

test-verbose:
	dotnet test --verbosity normal

test-tutorial:
	dotnet test tutorial/CanopyPlugin.Tutorial.csproj --logger "console;verbosity=detailed"

test-cov:
	dotnet test --collect:"XPlat Code Coverage" --results-directory:./TestResults

# Code quality
lint:
	dotnet format --verify-no-changes

format:
	dotnet format

# Protobuf generation
proto:
	dotnet-grpc refresh

# Clean
clean:
	dotnet clean
	rm -rf bin/
	rm -rf obj/
	rm -rf TestResults/
	rm -rf .vs/

# Plugin execution
run:
	dotnet run

run-plugin:
	dotnet run

# Development servers
serve:
	dotnet run --urls="http://0.0.0.0:8000"

serve-dev:
	dotnet watch run --urls="http://0.0.0.0:8000"

# Full validation
validate: format lint test

# Documentation
docs:
	@echo "Documentation available in README.md"
	@echo "API docs available at http://localhost:8000/docs when server is running"
