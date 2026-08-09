#!/usr/bin/env bash
# go2dgame 构建脚本：本机 (Termux/Linux) + 可选 Linux amd64 交叉编译
# 引擎依赖 X11（纯 Go 实现，零 cgo），运行需 termux-x11/XFCE 或 Linux 桌面。
# 用法：bash build.sh [cross]
#   （无参数）只构建本机三个程序
#   bash build.sh cross   额外交叉编译 Linux amd64 版（可拷到电脑上跑）
set -e
cd "$(dirname "$0")"
mkdir -p bin

echo "==> [1/1] 构建本机 ($(go env GOOS)/$(go env GOARCH)) ..."
CGO_ENABLED=0 go build -trimpath -o bin/pong  ./cmd/pong
CGO_ENABLED=0 go build -trimpath -o bin/smoke ./cmd/smoke
CGO_ENABLED=0 go build -trimpath -o bin/xinfo ./cmd/xinfo

if [ "$1" = "cross" ]; then
  echo "==> [2/2] 交叉编译 Linux amd64 ..."
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/pong-linux-amd64  ./cmd/pong
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/smoke-linux-amd64 ./cmd/smoke
fi

echo ""
echo "构建完成："
ls -lh bin/
