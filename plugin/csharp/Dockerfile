# Dockerfile for Canopy + C# Plugin
# Build from repository root: docker build -f plugin/csharp/Dockerfile -t canopy-csharp .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build C# Plugin (using Debian-based image for glibc compatibility with grpc.tools)
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS csharp-builder
WORKDIR /app
COPY plugin/csharp/ ./
RUN dotnet publish CanopyPlugin.csproj -c Release -o /out

# Stage 3: Runtime
FROM mcr.microsoft.com/dotnet/runtime:9.0
WORKDIR /app
RUN mkdir -p /tmp/plugin /root/.canopy

COPY --from=canopy-builder /canopy .
COPY --from=csharp-builder /out ./plugin/

# Set plugin type for Canopy to start
RUN printf '{"plugin":"csharp"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: dotnet plugin/CanopyPlugin.dll)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
