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
RUN mkdir -p /tmp/plugin /root/.canopy

COPY --from=canopy-builder /canopy .
COPY plugin/python/ ./plugin/

# Install dependencies if requirements.txt exists
RUN if [ -f plugin/requirements.txt ]; then pip install --no-cache-dir -r plugin/requirements.txt; fi

# Set plugin type for Canopy to start
ENV CANOPY_PLUGIN=python

# Default: run canopy (start plugin separately with: python plugin/main.py)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
