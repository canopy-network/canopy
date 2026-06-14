#!/usr/bin/env bash

set -eo pipefail

export PATH="$PATH:$HOME/go/bin:$HOME/bin"

~/bin/protoc29 -I=./ -I=/tmp/protoc29/include --go_out=../. --go_opt=module=github.com/canopy-network/canopy/plugin/go ./*.proto

find ../. -name "*.pb.go" | xargs -I {} protoc-go-inject-tag -input="{}"
