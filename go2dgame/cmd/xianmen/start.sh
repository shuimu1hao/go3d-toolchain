#!/data/data/com.termux/files/usr/bin/bash
# ============================================================
#  仙门物语 · 桌面版启动脚本（termux-x11 + XFCE 图形环境）
#  用法:
#    bash ~/hermes11/go3d-toolchain/go2dgame/cmd/xianmen/start.sh          启动游戏
#    bash ~/hermes11/go3d-toolchain/go2dgame/cmd/xianmen/start.sh stop     停止游戏
#    bash ~/hermes11/go3d-toolchain/go2dgame/cmd/xianmen/start.sh build    重新编译
#    bash ~/hermes11/go3d-toolchain/go2dgame/cmd/xianmen/start.sh bot [0|1|2] 自动测试某条线
# ============================================================
PROJ=~/hermes11/go3d-toolchain/go2dgame/cmd/xianmen
BIN=~/hermes11/go3d-toolchain/go2dgame/bin/xianmen

cd "$PROJ" || { echo "❌ 项目目录不存在"; exit 1; }

# 确保 DISPLAY（termux-x11 桌面）
if [ -z "$DISPLAY" ]; then
  export DISPLAY=:0
fi

# X 是否可用
if [ ! -S "$PREFIX/tmp/.X11-unix/X0" ]; then
  echo "❌ 桌面环境未运行！先执行: bash ~/hermes11/start-x11.sh"
  exit 1
fi

case "$1" in
  stop)
    pids=$(ps -eo pid,args | grep "[b]in/xianmen\|xianmen$" | grep -v hermes-snap | grep -v start.sh | awk '{print $1}')
    if [ -n "$pids" ]; then
      kill $pids 2>/dev/null; sleep 1
      echo "✅ 游戏已停止"
    else
      echo "ℹ️ 游戏未在运行"
    fi
    exit 0
    ;;
  build)
    cd ~/hermes11/go3d-toolchain/go2dgame && go build -o bin/xianmen ./cmd/xianmen && echo "✅ 编译完成: $BIN"
    exit 0
    ;;
  bot)
    # 自动测试：--bot=线号 (0师傅 1师姐 2师妹)，3 次通关后自动退出
    line="${2:-0}"
    if ps -eo pid,args | grep "[b]in/xianmen" | grep -v hermes-snap | grep -v start.sh >/dev/null; then
      echo "⚠️ 游戏已在运行，先 stop 再测"; exit 1
    fi
    cd "$PROJ" && timeout 120 "$BIN" --bot="$line" 2>&1 | grep -v XGB | grep -E "通关|结局|assets|font"
    echo "（测试完成）"
    exit 0
    ;;
esac

# 默认：启动
if ps -eo pid,args | grep "[b]in/xianmen" | grep -v hermes-snap | grep -v start.sh >/dev/null; then
  echo "✅ 游戏已在运行（如需重启先执行 stop）"
  exit 0
fi
echo "🚀 启动仙门物语桌面版..."
cd "$PROJ" && "$BIN" > "$PREFIX/tmp/xianmen_desktop.log" 2>&1 &
sleep 2
if ps -eo pid,args | grep "[b]in/xianmen" | grep -v hermes-snap | grep -v start.sh >/dev/null; then
  echo "✅ 游戏已启动！操作：Enter/空格/点击=推进，W/S=选选项"
  echo "   日志: $PREFIX/tmp/xianmen_desktop.log"
else
  echo "❌ 启动失败，日志:"; tail -5 "$PREFIX/tmp/xianmen_desktop.log" | grep -v XGB
fi
