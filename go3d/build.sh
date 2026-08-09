#!/usr/bin/env bash
# go3d 构建脚本：构建 demo + 运行测试。
set -e
cd "$(dirname "$0")"

echo "==> go vet"
go vet ./... 2>&1 | grep -v "unkeyed" || true

echo "==> go test"
go test ./...

echo "==> build demo3d"
mkdir -p bin
go build -o bin/demo3d ./cmd/demo3d

echo "==> done: ./bin/demo3d"
