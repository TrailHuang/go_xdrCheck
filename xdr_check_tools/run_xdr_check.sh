#!/bin/bash
# XDR检查工具启动脚本

# 检测系统架构
ARCH=$(uname -m)

case "$ARCH" in
    "x86_64")
        BINARY="linux/amd64/xdr_check_optimized"
        ;;
    "aarch64")
        BINARY="linux/arm64/xdr_check_optimized"
        ;;
    *)
        echo "不支持的架构: $ARCH"
        echo "支持的架构: x86_64, aarch64"
        exit 1
        ;;
esac

# 检查二进制文件是否存在
if [ ! -f "$BINARY" ]; then
    echo "错误: 找不到二进制文件 $BINARY"
    exit 1
fi

# 运行程序
chmod +x "$BINARY"
./"$BINARY" "$@"
