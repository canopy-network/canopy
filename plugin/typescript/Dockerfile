# Dockerfile for Canopy + TypeScript Plugin
# Build from repository root: docker build -f plugin/typescript/Dockerfile -t canopy-typescript .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build TypeScript Plugin
FROM node:20-alpine AS typescript-builder
WORKDIR /app
COPY plugin/typescript/ ./
RUN npm ci && npm run build

# Stage 3: Runtime
FROM node:20-alpine
WORKDIR /app
RUN mkdir -p /tmp/plugin /root/.canopy

COPY --from=canopy-builder /canopy .
COPY --from=typescript-builder /app/dist ./plugin/
COPY --from=typescript-builder /app/node_modules ./plugin/node_modules/
COPY --from=typescript-builder /app/package.json ./plugin/

# Set plugin type for Canopy to start
RUN printf '{"plugin":"typescript"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: node plugin/main.js)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
