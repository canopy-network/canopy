# Dockerfile for Canopy + Python Plugin
# Build from repository root: docker build -f plugin/python/Dockerfile -t canopy-python .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Runtime with Python
FROM python:3.12-alpine
WORKDIR /app

# Install bash (required for pluginctl.sh)
RUN apk add --no-cache bash

RUN mkdir -p /tmp/plugin /root/.canopy

# Copy Canopy binary
COPY --from=canopy-builder /canopy .

# Copy Python plugin to correct path (Canopy expects plugin/python/pluginctl.sh)
COPY plugin/python/ ./plugin/python/

# Remove any existing venv from copy, create fresh venv, and install dependencies
RUN rm -rf /app/plugin/python/.venv && \
    python3 -m venv /app/plugin/python/.venv && \
    /app/plugin/python/.venv/bin/pip install --no-cache-dir --upgrade pip && \
    /app/plugin/python/.venv/bin/pip install --no-cache-dir -e /app/plugin/python/

# Set plugin type for Canopy to start
RUN printf '{"plugin":"python"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: python plugin/main.py)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
