FROM golang:1.23.9-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

ARG EXPLORER_BASE_PATH
ARG WALLET_BASE_PATH
ARG BIN_PATH

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

ENV EXPLORER_BASE_PATH=${EXPLORER_BASE_PATH}
ENV WALLET_BASE_PATH=${WALLET_BASE_PATH}

RUN make build/wallet
RUN make build/explorer
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bin ./auto-update/.

# Only build if the file at ${BIN_PATH} doesn't already exist
RUN if [ ! -f "${BIN_PATH}" ]; then \
    echo "File ${BIN_PATH} not found. Building it..."; \
    CGO_ENABLED=0 GOOS=linux go build -a -o "${BIN_PATH}" ./cmd/main/...; \
  else \
    echo "File ${BIN_PATH} already exists. Skipping build."; \
  fi

FROM alpine:3.19

ARG BIN_PATH

WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./
COPY --from=builder /go/src/github.com/canopy-network/canopy/${BIN_PATH} ${BIN_PATH}
RUN chmod +x ${BIN_PATH}
ENTRYPOINT ["/app/bin"]
