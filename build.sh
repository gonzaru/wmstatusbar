#!/bin/sh

set -eu

mkdir -p bin
version=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
go build -ldflags="-s -w -X main.version=${version}" \
         -o bin/wmstatusbar ./cmd/wmstatusbar
