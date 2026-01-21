#!/bin/bash

# 打包脚本：只打包项目必要文件和目录
# 用法: ./package.sh [platform]
# platform: linux-amd64, linux-arm, darwin-arm64 (默认: linux-amd64)

PLATFORM=${1:-linux-amd64}
BINARY_NAME="GdbAlarm-${PLATFORM}"
PACKAGE_NAME="GdbAlarm-${PLATFORM}.tar.gz"

echo "打包平台: $PLATFORM"
echo "二进制文件: $BINARY_NAME"
echo "包名: $PACKAGE_NAME"

# 检查二进制文件是否存在
if [ ! -f "$BINARY_NAME" ]; then
    echo "错误: 二进制文件 $BINARY_NAME 不存在，请先编译。"
    exit 1
fi

# 检查必要目录
if [ ! -d "config" ]; then
    echo "错误: config 目录不存在。"
    exit 1
fi

# 创建临时打包目录
TEMP_DIR="temp_package"
mkdir -p "$TEMP_DIR"

# 复制必要文件
cp "$BINARY_NAME" "$TEMP_DIR/"
mkdir -p "$TEMP_DIR/config"
cp -r "config/alarm_filter.json" "$TEMP_DIR/config/"
cp -r "config/amp_api.yaml" "$TEMP_DIR/config/"
cp -r "config/mds.json" "$TEMP_DIR/config/"
cp "manager.sh" "$TEMP_DIR/" 2>/dev/null || echo "警告: manager.sh 不存在，跳过。"

# 创建 log 目录（空目录）
mkdir -p "$TEMP_DIR/log"

# 打包压缩
tar -czf "$PACKAGE_NAME" -C "$TEMP_DIR" .

# 清理临时目录
rm -rf "$TEMP_DIR"

echo "打包完成: $PACKAGE_NAME"
echo "包含文件:"
tar -tzf "$PACKAGE_NAME"