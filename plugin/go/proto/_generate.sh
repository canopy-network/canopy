#!/usr/bin/env bash

set -eo pipefail

export PATH="$PATH:$HOME/go/bin"

protoc -I=./ -I=/usr/include --go_out=../. --go_opt=module=github.com/ARBOR-L/ARBOR/plugin/go ./*.proto

find ../. -name "*.pb.go" | xargs -I {} protoc-go-inject-tag -input="{}"
