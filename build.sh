#!/bin/bash
# 一键构建：构建前端 -> 同步到后端 html/dist -> Go 编译单二进制（前端已内嵌）
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=========================================="
echo "  个人图书管理系统 - 生产构建"
echo "=========================================="

echo ""
echo "[1/3] 构建前端 (vite build)..."
cd "$PROJECT_DIR/frontend"
npm run build

echo ""
echo "[2/3] 同步前端产物到 backend/html/dist ..."
rm -rf "$PROJECT_DIR/backend/html/dist"
cp -r "$PROJECT_DIR/frontend/dist" "$PROJECT_DIR/backend/html/dist"

echo ""
echo "[3/3] 编译后端单二进制 (go build)..."
cd "$PROJECT_DIR/backend"
CGO_ENABLED=0 go build -o moyue-server .

echo ""
echo "=========================================="
echo "  构建完成：$PROJECT_DIR/backend/moyue-server"
echo "  运行：./backend/moyue-server"
echo "  访问：http://localhost:18080"
echo "=========================================="
