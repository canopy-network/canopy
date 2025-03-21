#!/usr/bin/env bash

set -eo pipefail

protoc -I=./ --go_out=../../. --go_opt=module=github.com/canopy-network/canopy ./*.proto

find ../../. -name "*.pb.go" | xargs -I {} protoc-go-inject-tag -input="{}"
