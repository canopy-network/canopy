.PHONY: restore build test test-tutorial lint format clean proto run serve serve-dev validate docs

# Development setup
restore:
	dotnet restore

# Build
build:
	dotnet build

build-release:
	dotnet build --configuration Release

# Build self-contained for local development (same as root Makefile)
# This builds a native executable that doesn't require .NET runtime
build-local:
	rm -rf bin && dotnet publish -c Release -r linux-x64 --self-contained true -o bin

# Testing
# Build first, then run tests with --no-build --no-restore so the test step
# never triggers an implicit NuGet restore against the network (which can hang
# when nuget.org is slow or unreachable). Packages are restored during `build`.
test: build
	dotnet test --no-build --no-restore

test-verbose: build
	dotnet test --no-build --no-restore --verbosity normal

test-tutorial: build
	dotnet test tutorial/CanopyPlugin.Tutorial.csproj --no-build --no-restore --logger "console;verbosity=detailed"

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
