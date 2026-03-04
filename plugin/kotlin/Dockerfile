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
RUN apk add --no-cache bash && mkdir -p /tmp/plugin /root/.canopy plugin/kotlin/build/libs

COPY --from=canopy-builder /canopy .
COPY --from=kotlin-builder /app/build/libs/*-all.jar plugin/kotlin/build/libs/canopy-plugin-kotlin-1.0.0-all.jar
COPY plugin/kotlin/pluginctl.sh plugin/kotlin/pluginctl.sh
RUN chmod +x plugin/kotlin/pluginctl.sh

# Set plugin type for Canopy to start
RUN printf '{"plugin":"kotlin"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: java -jar plugin.jar)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
