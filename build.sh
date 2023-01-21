#!/bin/bash

set -e

declare PROTO_DIR="./protobuf"
declare BIN="./med"

rm -f "$BIN"

protoc -I="$PROTO_DIR" --go_out=. "$PROTO_DIR/med.proto"

# go build -ldflags="-w -s"  -o "$BIN" ./main.go
go build -o "$BIN" ./main.go
