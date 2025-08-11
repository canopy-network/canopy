FROM golang:1.24.0-alpine AS builder

RUN apk update && apk add --no-cache make bash git nodejs npm

# Set up build args and environment for canopy
ARG EXPLORER_BASE_PATH
ARG WALLET_BASE_PATH

# Clone and build go-plugin
WORKDIR /go/src/github.com/canopy-network
RUN git clone https://github.com/canopy-network/go-plugin.git

WORKDIR /go/src/github.com/canopy-network/go-plugin
RUN make build

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

ENV EXPLORER_BASE_PATH=${EXPLORER_BASE_PATH}
ENV WALLET_BASE_PATH=${WALLET_BASE_PATH}

# Build canopy binaries
RUN make build/wallet
RUN make build/explorer
RUN go build -a -o /build/canopy ./cmd/main/...

# --- Final image ---
FROM alpine:3.19
WORKDIR /app

# Copy binaries
COPY --from=builder /build/canopy /app/canopy
COPY --from=builder /root/go/bin/go-plugin /app/plugin

# Plugin manages process coordination, just run both
CMD /app/canopy start & /app/plugin