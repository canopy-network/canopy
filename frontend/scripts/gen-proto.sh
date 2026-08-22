#!/usr/bin/env bash
set -euo pipefail

PROTO_DIR="proto"
OUT_DIR="src/lib/canopy/proto/generated"

mkdir -p "$OUT_DIR"

# Termux well-known types location
WELL_KNOWN_INCLUDE="/data/data/com.termux/files/usr/include"

if [ ! -f "$WELL_KNOWN_INCLUDE/google/protobuf/any.proto" ]; then
  WELL_KNOWN_INCLUDE="/usr/include"
fi

if [ ! -f "$WELL_KNOWN_INCLUDE/google/protobuf/any.proto" ]; then
  WELL_KNOWN_INCLUDE="/usr/local/include"
fi

echo "Proto dir: $PROTO_DIR"
echo "Output dir: $OUT_DIR"
echo "Well-known include: $WELL_KNOWN_INCLUDE"

protoc \
  --plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto \
  --ts_proto_out="$OUT_DIR" \
  --ts_proto_opt=esModuleInterop=true \
  --ts_proto_opt=forceLong=bigint \
  --ts_proto_opt=env=browser \
  --ts_proto_opt=outputEncodeMethods=true \
  --ts_proto_opt=outputJsonMethods=true \
  --ts_proto_opt=outputClientImpl=false \
  --ts_proto_opt=useDate=false \
  --ts_proto_opt=onlyTypes=false \
  --ts_proto_opt=unrecognizedEnum=false \
  -I"$PROTO_DIR" \
  -I"$WELL_KNOWN_INCLUDE" \
  "$PROTO_DIR"/*.proto \
  "$WELL_KNOWN_INCLUDE/google/protobuf/any.proto"

echo "Proto generation complete."
