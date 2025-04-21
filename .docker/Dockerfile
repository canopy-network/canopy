FROM golang:1.22.1-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

ARG BUILD_PATH
ARG EXPLORER_BASE_PATH
ARG WALLET_BASE_PATH

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

ENV EXPLORER_BASE_PATH=${EXPLORER_BASE_PATH}
ENV WALLET_BASE_PATH=${WALLET_BASE_PATH}

RUN make build/wallet
RUN make build/explorer
RUN go build -a -o bin ./${BUILD_PATH}/*.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./
ENTRYPOINT ["/app/bin"]
