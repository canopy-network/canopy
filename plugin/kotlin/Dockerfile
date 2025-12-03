# Dockerfile for Canopy + Kotlin Plugin
# Build from repository root: docker build -f plugin/kotlin/Dockerfile -t canopy-kotlin .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build Kotlin Plugin
FROM eclipse-temurin:21-jdk-alpine AS kotlin-builder
WORKDIR /app
COPY plugin/kotlin/ ./
RUN ./gradlew fatJar --no-daemon

# Stage 3: Runtime
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
RUN mkdir -p /tmp/plugin /root/.canopy

COPY --from=canopy-builder /canopy .
COPY --from=kotlin-builder /app/build/libs/*-all.jar plugin.jar

# Set plugin type for Canopy to start
ENV CANOPY_PLUGIN=kotlin

# Default: run canopy (start plugin separately with: java -jar plugin.jar)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
