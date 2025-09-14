FROM golang:1.23.9-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy

RUN make build/wallet
RUN make build/explorer
RUN go build -a -o bin ./cmd/main/...

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./
ENTRYPOINT ["/app/bin"]
