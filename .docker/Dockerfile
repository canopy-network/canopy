FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

# Wallet-specific build configuration
# Default: / (for simple-stack deployment with direct port access)
ARG VITE_WALLET_BASE_PATH=/
# RPC proxy targets for chain.json generation
ARG VITE_WALLET_RPC_PROXY_TARGET=http://localhost:50002
ARG VITE_WALLET_ADMIN_RPC_PROXY_TARGET=http://localhost:50003

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

# Export build configuration to environment
ENV VITE_WALLET_BASE_PATH=${VITE_WALLET_BASE_PATH}
ENV VITE_WALLET_RPC_PROXY_TARGET=${VITE_WALLET_RPC_PROXY_TARGET}
ENV VITE_WALLET_ADMIN_RPC_PROXY_TARGET=${VITE_WALLET_ADMIN_RPC_PROXY_TARGET}

RUN make build/wallet
RUN make build/explorer
RUN go build -a -o bin ./cmd/main/...

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./
ENTRYPOINT ["/app/bin"]
