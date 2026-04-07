#!/bin/bash

# XDR检查工具构建脚本
# 同时编译Linux x86_64和ARM64版本并打包

PKT_VER="go_xdr_check"
PLATFORM="linux"
ARCH="x86_64"
release_dir="xdr_check_tools"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "命令 $1 未找到，请先安装"
        exit 1
    fi
}

# 检查环境
check_environment() {
    log_info "检查构建环境..."

    # 检查必要的命令
    check_command "go"
    check_command "make"
    check_command "tar"

    # 检查Makefile是否存在
    if [ ! -f "Makefile" ]; then
        log_error "Makefile不存在，请确保在项目根目录运行此脚本"
        exit 1
    fi

    # 获取平台和架构
    if command -v "winver" &> /dev/null; then
        PLATFORM="windows"
    else
        PLATFORM="linux"
    fi

    ARCH=$(uname -m)

    log_info "平台: $PLATFORM"
    log_info "架构: $ARCH"
}

# 安装依赖
install_dependencies() {
    log_info "安装Go模块依赖..."
    if ! make install-deps; then
        log_error "依赖安装失败"
        exit 1
    fi
    log_success "依赖安装完成"
}

# 编译项目 - 同时编译x86和ARM版本
build_project() {
    log_info "开始编译项目..."

    # 编译Linux x86_64版本
    log_info "编译Linux x86_64版本..."
    if ! make build-linux; then
        log_error "Linux x86_64版本编译失败"
        exit 1
    fi

    # 编译Linux ARM64版本
    log_info "编译Linux ARM64版本..."
    if ! make build-linux-arm64; then
        log_error "Linux ARM64版本编译失败"
        exit 1
    fi

    # 检查构建是否成功
    if [ ! -f "build/bin/linux/amd64/xdr_check_optimized" ]; then
        log_error "Linux x86_64版本构建失败"
        exit 1
    fi

    if [ ! -f "build/bin/linux/arm64/xdr_check_optimized" ]; then
        log_error "Linux ARM64版本构建失败"
        exit 1
    fi

    log_success "项目编译完成（x86_64和ARM64版本）"
}

# 创建发布包 - 包含x86和ARM版本
create_release_package() {
    log_info "创建发布包..."

    # 获取版本信息
    VERSION="v1.1.18"

    # 创建发布目录
    rm -rf "$release_dir"
    mkdir -p "$release_dir"

    # 创建架构子目录
    mkdir -p "$release_dir/linux/amd64"
    mkdir -p "$release_dir/linux/arm64"

    # 复制x86_64版本二进制文件
    cp -f "build/bin/linux/amd64/xdr_check_optimized" "$release_dir/linux/amd64/"

    # 复制ARM64版本二进制文件
    cp -f "build/bin/linux/arm64/xdr_check_optimized" "$release_dir/linux/arm64/"

    # 复制配置文件
    cp -f *.ini "$release_dir/" 2>/dev/null || true
    cp -f *.txt "$release_dir/" 2>/dev/null || true
    cp -f conf "$release_dir/" 2>/dev/null || true
    cp -f xdr_check_template-*.xlsx "$release_dir/" 2>/dev/null || true

    # 设置默认配置文件
    if [ -f "xdr_check-IDC.ini" ]; then
        cp -f "xdr_check-IDC.ini" "$release_dir/xdr_check.ini"
    fi

    # 创建版本信息文件
    cat > "$release_dir/VERSION" << EOF
XDR Check Tool
Version: $VERSION
Platform: Linux
Supported Architectures: x86_64, ARM64
Build Time: $(date "+%Y-%m-%d %H:%M:%S")

Usage:
- For x86_64 systems: ./linux/amd64/xdr_check_optimized
- For ARM64 systems: ./linux/arm64/xdr_check_optimized
EOF

    # 创建启动脚本
    cat > "$release_dir/run_xdr_check.sh" << 'EOF'
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
EOF

    chmod +x "$release_dir/run_xdr_check.sh"

    # 打包
    PACKAGE_NAME="${PKT_VER}_linux_multiarch_${VERSION}"

    # 确保包名中没有特殊字符
    PACKAGE_NAME=$(echo "$PACKAGE_NAME" | tr -d '[:space:]')

    # 创建压缩包
    tar czvf "${PACKAGE_NAME}.tar.gz" "$release_dir"

    if [ $? -eq 0 ]; then
        log_success "发布包创建完成: ${PACKAGE_NAME}.tar.gz"
        log_info "包含架构: x86_64, ARM64"
    else
        log_error "打包失败"
        exit 1
    fi
}

# 清理工作
cleanup() {
    log_info "清理临时文件..."
    # 可以选择保留发布目录用于调试
    # rm -rf "$release_dir"
    log_success "清理完成"
}

# 显示帮助信息
show_help() {
    echo "XDR检查工具构建脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -c, --clean    构建前先清理"
    echo "  -t, --test     构建后运行测试"
    echo ""
    echo "说明:"
    echo "  此脚本会同时编译Linux x86_64和ARM64版本，并打包成一个发布包"
    echo "  发布包中包含适用于两种架构的二进制文件和启动脚本"
    echo ""
    echo "示例:"
    echo "  $0              # 构建多架构版本"
    echo "  $0 -c           # 清理后构建"
    echo "  $0 -t           # 构建后运行测试"
}

# 主函数
main() {
    local clean_build=false
    local run_tests=false

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--clean)
                clean_build=true
                shift
                ;;
            -t|--test)
                run_tests=true
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 检查环境
    check_environment

    # 清理构建（如果指定）
    if [ "$clean_build" = true ]; then
        log_info "执行清理..."
        make clean
    fi

    # 安装依赖
    install_dependencies

    # 构建项目
    build_project

    # 运行测试（如果指定）
    if [ "$run_tests" = true ]; then
        log_info "运行测试..."
        make test
    fi

    # 创建发布包
    create_release_package

    # 清理
    cleanup

    log_success "XDR检查工具多架构构建完成！"
    log_info "发布包包含: Linux x86_64 和 ARM64 版本"
}

# 运行主函数
main "$@"