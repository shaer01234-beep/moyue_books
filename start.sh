#!/bin/bash
# 个人图书管理系统启动脚本

set -e

echo "=========================================="
echo "  个人图书管理系统 - 启动脚本"
echo "=========================================="

# 检测必要命令
command -v go >/dev/null 2>&1 || { echo "错误：需要安装 Go"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "错误：需要安装 Node.js/npm"; exit 1; }

# 项目根目录
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 启动后端
echo ""
echo "[1/3] 正在编译并启动后端服务 (localhost:18080)..."
cd "$PROJECT_DIR/backend"
CGO_ENABLED=0 go build -o moyue-server .
./moyue-server &
BACKEND_PID=$!
echo "      后端 PID: $BACKEND_PID"

# 等待后端启动
sleep 2

# 启动前端
echo ""
echo "[2/3] 正在启动前端开发服务器 (localhost:3000)..."
cd "$PROJECT_DIR/frontend"
if [ ! -d node_modules ]; then
    echo "      首次安装前端依赖..."
    npm install
fi
npm run dev &
FRONTEND_PID=$!
echo "      前端 PID: $FRONTEND_PID"

# 等待前端启动
sleep 3

echo ""
echo "[3/3] 启动完成！"
echo ""
echo "  🌐 前端地址: http://localhost:3000"
echo "  🔌 API 地址: http://localhost:18080"
echo ""
echo "  按 Ctrl+C 停止服务"
echo "=========================================="

# 捕获中断信号
trap "echo ''; echo '正在关闭服务...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit 0" INT TERM

wait
