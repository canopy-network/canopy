# Dockerfile for Canopy + C# Plugin
# Build from repository root: docker build -f plugin/csharp/Dockerfile -t canopy-csharp .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build C# Plugin (using Debian SDK for grpc.tools compatibility, target musl for Alpine runtime)
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS csharp-builder
WORKDIR /app
COPY plugin/csharp/ ./
# Build self-contained for Alpine (musl) - must use Debian SDK because grpc.tools protoc is glibc-only
RUN dotnet publish CanopyPlugin.csproj -c Release -r linux-musl-x64 --self-contained true -o /out

# Stage 3: Runtime (Alpine-based, self-contained binary includes .NET runtime)
FROM alpine:3.19
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache bash procps libstdc++ libgcc icu-libs

RUN mkdir -p /tmp/plugin /root/.canopy plugin/csharp/bin

COPY --from=canopy-builder /canopy .
COPY --from=csharp-builder /out ./plugin/csharp/bin/

# Copy pluginctl.sh from source
COPY plugin/csharp/pluginctl.sh ./plugin/csharp/pluginctl.sh
RUN chmod +x plugin/csharp/pluginctl.sh

# Set plugin type for Canopy to start
RUN printf '{"plugin":"csharp"}\n' > /root/.canopy/config.json

# Default: run canopy
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
