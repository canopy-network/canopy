# Dockerfile for Canopy + Go Plugin
# Build from repository root: docker build -f plugin/go/Dockerfile -t canopy-go .

# Stage 1: Build Canopy
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build Go Plugin
FROM golang:1.24-alpine AS plugin-builder
WORKDIR /app
COPY plugin/go/ ./
RUN go build -o /plugin .

# Stage 3: Runtime
FROM alpine:latest
WORKDIR /app
RUN mkdir -p /tmp/plugin /root/.canopy

COPY --from=canopy-builder /canopy .
COPY --from=plugin-builder /plugin .

# Set plugin type for Canopy to start
RUN printf '{"plugin":"go"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: ./plugin)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
