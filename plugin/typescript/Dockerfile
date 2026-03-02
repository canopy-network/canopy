# Dockerfile for Canopy + TypeScript Plugin
# Build from repository root: docker build -f plugin/typescript/Dockerfile -t canopy-typescript .

# Stage 1: Build Canopy (Go)
FROM golang:1.25-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build TypeScript Plugin
FROM node:20-alpine AS typescript-builder
WORKDIR /app
COPY plugin/typescript/ ./
RUN npm ci && npm run build:all

# Stage 3: Runtime
FROM node:20-alpine
WORKDIR /app
RUN apk add --no-cache bash && mkdir -p /tmp/plugin /root/.canopy plugin/typescript

COPY --from=canopy-builder /canopy .
COPY --from=typescript-builder /app/dist plugin/typescript/dist/
COPY --from=typescript-builder /app/node_modules plugin/typescript/node_modules/
COPY --from=typescript-builder /app/package.json plugin/typescript/
COPY plugin/typescript/pluginctl.sh plugin/typescript/pluginctl.sh
RUN chmod +x plugin/typescript/pluginctl.sh

# Set plugin type for Canopy to start
RUN printf '{"plugin":"typescript"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: node plugin/main.js)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
