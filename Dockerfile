FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

ARG BIN_PATH
# Wallet-specific build configuration
# Default: /wallet/ (for monitoring-stack deployment with Traefik reverse proxy)
# Override with: docker build --build-arg VITE_WALLET_BASE_PATH=/
ARG VITE_WALLET_BASE_PATH=/wallet/
# RPC proxy targets for chain.json generation
# For monitoring-stack, these should be Traefik URLs
ARG VITE_WALLET_RPC_PROXY_TARGET=/wallet/rpc
ARG VITE_WALLET_ADMIN_RPC_PROXY_TARGET=/wallet/adminrpc

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

# Export build configuration to environment
# These are available during npm build for wallet and explorer
ENV VITE_WALLET_BASE_PATH=${VITE_WALLET_BASE_PATH}
ENV VITE_WALLET_RPC_PROXY_TARGET=${VITE_WALLET_RPC_PROXY_TARGET}
ENV VITE_WALLET_ADMIN_RPC_PROXY_TARGET=${VITE_WALLET_ADMIN_RPC_PROXY_TARGET}

RUN make build/wallet
RUN make build/explorer
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bin ./cmd/auto-update/.

# Only build if the file at ${BIN_PATH} doesn't already exist
RUN if [ ! -f "${BIN_PATH}" ]; then \
  echo "File ${BIN_PATH} not found. Building it..."; \
  CGO_ENABLED=0 GOOS=linux go build -a -o "${BIN_PATH}" ./cmd/main/...; \
  else \
  echo "File ${BIN_PATH} already exists. Skipping build."; \
  fi

FROM alpine:3.19

RUN apk add --no-cache pigz ca-certificates

ARG BIN_PATH

WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./
COPY --from=builder /go/src/github.com/canopy-network/canopy/${BIN_PATH} ${BIN_PATH}
RUN chmod +x ${BIN_PATH}
ENTRYPOINT ["/app/bin"]
