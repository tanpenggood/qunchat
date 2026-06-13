#!/bin/bash
set -e

BACKEND="${1:-}"

if [ -z "$BACKEND" ]; then
  echo "=== Qunchat 后端选择 ==="
  echo ""
  echo "  1) Go"
  echo "  2) Node"
  echo "  3) Python"
  echo ""
  read -p "输入编号 (默认 1): " BACKEND
  BACKEND=${BACKEND:-1}
fi

case $BACKEND in
  1|go)
    cmd="go"
    name="Go"
    run="go run ./server"
    ;;
  2|node)
    cmd="node"
    name="Node"
    run="node scripts/dev.js"
    ;;
  3|python)
    cmd="python3"
    name="Python"
    run="python3 scripts/server.py"
    ;;
  *)
    echo "无效选项: $BACKEND"
    exit 1
    ;;
esac

if ! command -v "$cmd" &>/dev/null; then
  echo "错误: 未找到 $cmd，请先安装 $name 运行时"
  exit 1
fi

cd "$(dirname "$0")"
echo "启动 $name 后端 (http://localhost:${PORT:-8080})..."
exec $run
