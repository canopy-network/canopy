FROM golang:1.24-alpine AS builder

run apk update && apk add --no-cache make bash nodejs npm

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

RUN make build/wallet
RUN make build/explorer
RUN go build -a -o bin ./cmd/main/...

from alpine:3.19
workdir /app
copy --from=builder /go/src/github.com/canopy-network/canopy/bin ./
entrypoint ["/app/bin"]
