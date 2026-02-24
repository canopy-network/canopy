#!/usr/bin/env bash

set -eo pipefail

export PATH="$PATH:$HOME/go/bin"

protoc -I=./ -I=/usr/include --go_out=../. --go_opt=module=github.com/canopy-network/go-plugin ./*.proto

find ../. -name "*.pb.go" | xargs -I {} protoc-go-inject-tag -input="{}"
