#!/bin/bash

# GoldenDB 监控服务管理脚本
# 自动检测操作系统和架构，支持启动、停止、重启功能

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROGRAM_NAME="GdbAlarm"
PID_FILE="$SCRIPT_DIR/manager.pid"

# 获取当前操作系统和架构
get_platform_info() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            ARCH="unknown"
            ;;
    esac

    echo "${OS}-${ARCH}"
}

# 获取程序路径
get_program_path() {
    PLATFORM=$(get_platform_info)
    PROGRAM_PATH="$SCRIPT_DIR/${PROGRAM_NAME}-${PLATFORM}"

    if [ ! -f "$PROGRAM_PATH" ]; then
        # 如果没有平台特定的程序，尝试使用默认名称
        if [ -f "$SCRIPT_DIR/$PROGRAM_NAME" ]; then
            PROGRAM_PATH="$SCRIPT_DIR/$PROGRAM_NAME"
        else
            echo "错误: 找不到可执行程序" >&2
            exit 1
        fi
    fi

    echo "$PROGRAM_PATH"
}

# 检查服务是否在运行
is_running() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null 2>&1; then
            return 0
        else
            rm -f "$PID_FILE"
        fi
    fi
    return 1
}

# 启动服务
start_service() {
    if is_running; then
        echo "服务已经在运行 (PID: $(cat $PID_FILE))"
        return 0
    fi

    PROGRAM=$(get_program_path)
    echo "检测到平台: $(get_platform_info)"
    echo "启动程序: $PROGRAM"

    # 后台运行程序
    nohup "$PROGRAM" -s > /dev/null 2>&1 &
    SERVICE_PID=$!

    # 写入PID文件
    echo $SERVICE_PID > "$PID_FILE"

    echo "服务启动成功 (PID: $SERVICE_PID)"
}

# 停止服务
stop_service() {
    if ! is_running; then
        echo "服务未运行"
        return 0
    fi

    PID=$(cat "$PID_FILE")
    echo "正在停止服务 (PID: $PID)"

    # 发送SIGTERM信号
    kill $PID

    # 等待最多10秒
    COUNTER=0
    while [ $COUNTER -lt 10 ]; do
        if ! ps -p $PID > /dev/null 2>&1; then
            rm -f "$PID_FILE"
            echo "服务已停止"
            return 0
        fi
        sleep 1
        COUNTER=$((COUNTER + 1))
    done

    # 如果正常停止失败，强制杀死
    echo "正常停止超时，强制杀死进程"
    kill -9 $PID
    rm -f "$PID_FILE"
    echo "服务已强制停止"
}

# 重启服务
restart_service() {
    echo "重启服务..."
    stop_service
    sleep 2
    start_service
}

# 查看服务状态
status_service() {
    if is_running; then
        PID=$(cat "$PID_FILE")
        echo "服务正在运行 (PID: $PID)"
    else
        echo "服务未运行"
    fi
}

# 主逻辑
case "$1" in
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        restart_service
        ;;
    status)
        status_service
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status}"
        echo "检测到的平台: $(get_platform_info)"
        exit 1
        ;;
esac

exit 0