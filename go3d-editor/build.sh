#!/usr/bin/env bash
# go3d-editor 构建脚本。
set -e
cd "$(dirname "$0")"

echo "==> go vet"
go vet ./... 2>&1 | grep -v "unkeyed" || true

echo "==> go test"
go test ./...

echo "==> build"
mkdir -p bin
go build -o bin/go3d-editor ./cmd/go3d-editor

echo "==> done: ./bin/go3d-editor"
