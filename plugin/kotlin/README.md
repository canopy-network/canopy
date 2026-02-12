# Canopy Plugin - Kotlin Implementation

A Kotlin-based Canopy blockchain plugin that communicates with the Canopy FSM using protobuf messages over Unix domain sockets.

## Features

- **Type-safe configuration management** with data classes
- **Coroutine-based async networking** using Ktor
- **Protobuf message handling** with Google's protobuf-kotlin
- **Structured logging** with kotlin-logging
- **Comprehensive testing** with JUnit 5 and MockK

## Prerequisites

- JDK 17 or higher
- Gradle 8.0 or higher

## Project Structure

```
src/
├── main/
│   ├── kotlin/
│   │   └── com/canopy/plugin/
│   │       ├── Main.kt              # Application entry point
│   │       ├── config/
│   │       │   └── Config.kt        # Configuration management
│   │       ├── core/
│   │       │   └── Contract.kt      # Smart contract abstraction
│   │       ├── network/
│   │       │   └── SocketClient.kt  # Unix socket communication
│   │       └── utils/
│   │           └── Logger.kt        # Logging utilities
│   └── proto/                       # Protobuf definitions
└── test/
    └── kotlin/
        └── com/canopy/plugin/
            └── config/
                └── ConfigTest.kt     # Configuration tests
```

## Building

```bash
# Build the project
make build

# Or using Gradle directly
./gradlew build
```

## Running

```bash
# Run the plugin
make run

# Run in development mode
make dev

# Run with debugging on port 5005
make debug
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
./gradlew test jacocoTestReport
```

## Code Quality

```bash
# Run all validation checks (type check + tests)
make validate

# Type check only
make type-check
```

## Configuration

The plugin uses the following default configuration:

- **Chain ID**: 1
- **Data Directory**: `/tmp/plugin/`
- **Socket Path**: `/tmp/plugin/plugin.sock`

Configuration can be customized by:

1. Creating a JSON configuration file:
```json
{
  "chainId": 42,
  "dataDirPath": "/custom/path/"
}
```

2. Loading it in the application:
```kotlin
val config = Config.fromFile("/path/to/config.json")
```

### Debugging

The application supports remote debugging:

```bash
# Start with debug port 5005
make debug

# Connect your IDE debugger to localhost:5005
```

### Logging

Set log level via environment variable:

```bash
LOG_LEVEL=debug make run
```

Supported levels: `debug`, `info`, `warn`, `error`

## Protobuf

Proto files are compiled automatically during build. To manually generate:

```bash
make proto
```

Generated files are placed in `build/generated/source/proto/`

## Deployment

### Building JAR

```bash
# Standard JAR
make jar

# Fat JAR with all dependencies
make fatjar
```

### Docker (if needed)

```dockerfile
FROM openjdk:17-slim
COPY build/libs/canopy-plugin-kotlin-1.0.0-all.jar /app/plugin.jar
CMD ["java", "-jar", "/app/plugin.jar"]
```

## Troubleshooting

### Socket Connection Issues
- Ensure the FSM is running and the socket file exists
- Check permissions on the socket file
- Verify the data directory path is correct

### Memory Issues
- Adjust JVM heap size in gradle.properties
- Use `-Xmx` flag when running: `java -Xmx1024m -jar plugin.jar`

### Build Issues
- Clean and rebuild: `make clean build`
- Update dependencies: `./gradlew --refresh-dependencies`

## License

MIT
