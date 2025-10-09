# Canopy C# Plugin Development Guide

## Build & Test Commands
- **Build**: `make build` or `dotnet build`
- **Test**: `make test` or `dotnet test`
- **Test with coverage**: `make test-cov`
- **Lint**: `make lint` (verifies formatting)
- **Format**: `make format` (applies formatting)
- **Full validation**: `make validate` (format + lint + test)
- **Run plugin**: `make run` or `dotnet run`
- **Development server**: `make serve-dev` (with hot reload)

## Code Style Guidelines
- **Target**: .NET 8.0 with C# 12, nullable reference types enabled
- **Indentation**: 4 spaces for C#, 2 spaces for JSON/XML
- **Naming**: PascalCase for types/methods/properties, interfaces prefixed with `I`
- **Braces**: Allman style (new line before opening brace)
- **Imports**: Use `using` statements at top, prefer explicit types over `var` except when type is apparent
- **Error handling**: Custom exceptions inherit from `PluginException`, use structured logging
- **Async**: Prefer `async/await`, use `CancellationToken` for long-running operations
- **Null safety**: Leverage nullable reference types, use null-conditional operators
- **Logging**: Use `ILogger<T>` with structured logging and appropriate log levels
- **Configuration**: Use Microsoft.Extensions.Configuration pattern
- **Testing**: xUnit framework with Moq for mocking (when tests exist)

## Project Structure
- Core logic in `src/CanopyPlugin/core/`
- Socket communication in `src/CanopyPlugin/socket/`
- Protobuf definitions in `proto/`
- Configuration in `src/CanopyPlugin/config.cs`