from golang:1.23.9-alpine as builder

run apk update && apk add --no-cache make bash nodejs npm

arg explorer_base_path
arg wallet_base_path

workdir /go/src/github.com/canopy-network/canopy
copy . /go/src/github.com/canopy-network/canopy

env explorer_base_path=${explorer_base_path}
env wallet_base_path=${wallet_base_path}

#run make build/wallet
#run make build/explorer
run go build -a -o bin ./cmd/main/...

from alpine:3.19
workdir /app
copy --from=builder /go/src/github.com/canopy-network/canopy/bin ./
entrypoint ["/app/bin"]
