#!/bin/bash

# 抖音弹幕获取工具部署脚本

set -e

echo "🚀 开始部署抖音弹幕获取工具..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到Go环境，请先安装Go 1.19或更高版本"
    exit 1
fi

echo "✅ Go环境检查通过"

# 检查Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "📋 当前Go版本: $GO_VERSION"

# 安装依赖
echo "📦 安装依赖包..."
go mod tidy

# 构建项目
echo "🔨 构建项目..."
go build -o douyin-danmu-tool main.go

if [ $? -eq 0 ]; then
    echo "✅ 构建成功"
else
    echo "❌ 构建失败"
    exit 1
fi

# 设置执行权限
chmod +x douyin-danmu-tool

# 创建启动脚本
cat > start.sh << 'EOF'
#!/bin/bash

# 抖音弹幕获取工具启动脚本

PORT=${PORT:-12000}
HOST=${HOST:-0.0.0.0}

echo "🎭 启动抖音弹幕获取工具..."
echo "📍 服务地址: http://$HOST:$PORT"
echo "🔌 WebSocket地址: ws://$HOST:$PORT/ws"
echo ""
echo "按 Ctrl+C 停止服务"
echo ""

./douyin-danmu-tool -port $PORT -host $HOST
EOF

chmod +x start.sh

# 创建后台启动脚本
cat > start-daemon.sh << 'EOF'
#!/bin/bash

# 抖音弹幕获取工具后台启动脚本

PORT=${PORT:-12000}
HOST=${HOST:-0.0.0.0}
LOG_FILE=${LOG_FILE:-server.log}

# 检查是否已经在运行
if pgrep -f "douyin-danmu-tool" > /dev/null; then
    echo "⚠️  服务已在运行中"
    echo "如需重启，请先运行: ./stop.sh"
    exit 1
fi

echo "🎭 后台启动抖音弹幕获取工具..."
echo "📍 服务地址: http://$HOST:$PORT"
echo "🔌 WebSocket地址: ws://$HOST:$PORT/ws"
echo "📝 日志文件: $LOG_FILE"

nohup ./douyin-danmu-tool -port $PORT -host $HOST > $LOG_FILE 2>&1 &

sleep 2

if pgrep -f "douyin-danmu-tool" > /dev/null; then
    echo "✅ 服务启动成功"
    echo "📋 查看日志: tail -f $LOG_FILE"
    echo "⏹️  停止服务: ./stop.sh"
else
    echo "❌ 服务启动失败，请检查日志: $LOG_FILE"
    exit 1
fi
EOF

chmod +x start-daemon.sh

# 创建停止脚本
cat > stop.sh << 'EOF'
#!/bin/bash

# 抖音弹幕获取工具停止脚本

echo "⏹️  正在停止抖音弹幕获取工具..."

if pgrep -f "douyin-danmu-tool" > /dev/null; then
    pkill -f "douyin-danmu-tool"
    sleep 2
    
    if pgrep -f "douyin-danmu-tool" > /dev/null; then
        echo "⚠️  正常停止失败，强制终止..."
        pkill -9 -f "douyin-danmu-tool"
    fi
    
    echo "✅ 服务已停止"
else
    echo "ℹ️  服务未在运行"
fi
EOF

chmod +x stop.sh

# 创建状态检查脚本
cat > status.sh << 'EOF'
#!/bin/bash

# 抖音弹幕获取工具状态检查脚本

echo "📊 抖音弹幕获取工具状态检查"
echo "================================"

if pgrep -f "douyin-danmu-tool" > /dev/null; then
    PID=$(pgrep -f "douyin-danmu-tool")
    echo "✅ 服务状态: 运行中"
    echo "🆔 进程ID: $PID"
    
    # 检查端口占用
    if command -v netstat &> /dev/null; then
        PORT_INFO=$(netstat -tlnp 2>/dev/null | grep ":12000 " | head -1)
        if [ ! -z "$PORT_INFO" ]; then
            echo "🔌 端口状态: 12000端口已监听"
        fi
    fi
    
    # 显示内存使用
    if command -v ps &> /dev/null; then
        MEMORY=$(ps -p $PID -o rss= 2>/dev/null | awk '{print int($1/1024)"MB"}')
        echo "💾 内存使用: $MEMORY"
    fi
    
    echo "📝 查看日志: tail -f server.log"
    echo "⏹️  停止服务: ./stop.sh"
else
    echo "❌ 服务状态: 未运行"
    echo "🚀 启动服务: ./start.sh 或 ./start-daemon.sh"
fi

echo ""
echo "📍 访问地址: http://localhost:12000"
EOF

chmod +x status.sh

echo ""
echo "🎉 部署完成！"
echo ""
echo "📋 可用命令:"
echo "  ./start.sh          - 前台启动服务"
echo "  ./start-daemon.sh   - 后台启动服务"
echo "  ./stop.sh           - 停止服务"
echo "  ./status.sh         - 查看服务状态"
echo ""
echo "📍 访问地址: http://localhost:12000"
echo ""
echo "🚀 现在可以运行 ./start.sh 启动服务"