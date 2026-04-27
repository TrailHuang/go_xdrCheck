# Go XDR Check

基于Go语言重写的XDR（扩展检测与响应）日志检查工具，提供比Python版本更高的性能和更好的并发处理能力。

## 特性

- **高性能**: 使用Go语言编写，编译为原生二进制文件，执行效率更高
- **并发处理**: 内置goroutine并发机制，可同时检查多个目录
- **内存安全**: Go语言的垃圾回收机制避免内存泄漏
- **跨平台**: 支持Linux、Windows、macOS等多平台编译
- **兼容性**: 完全兼容原Python版本的配置文件和命令行参数

## 项目结构

```
go_xdrCheck/
├── main.go                 # 主程序入口
├── config/                 # 配置解析模块
│   └── config.go
├── checker/               # 文件检查模块
│   └── file_checker.go
├── validator/             # 数据校验模块
│   ├── ip_validator.go
│   └── rule_validator.go
├── parser/                # 模板解析模块
│   └── excel_parser.go
├── core/                  # 核心检查逻辑
│   └── checker.go
├── xdr_check.ini         # 配置文件
├── build.sh              # 构建脚本
└── README.md             # 说明文档
```

## 安装和使用

### 构建项目

```bash
# 下载依赖
go mod tidy

# 构建项目
./build.sh
```

### 命令行参数

```bash
# 帮助信息
./xdr_check -h

# 版本信息
./xdr_check -v

# 检查指定时间的日志
./xdr_check -t 20250101

# 检查当天日志
./xdr_check -o

# 抽样检查（每个目录检查10个文件）
./xdr_check -n 10

# 不检查子目录
./xdr_check -p
```

### 配置文件

Go版本完全兼容Python版本的配置文件格式：

```ini
[DEFAULT]
col_delimiter = |++|

[XDR_PATH]
xdr_template_file = xdr_check_template-AV.xlsx
EVT_.txt-fncheck = /home/data/udpi_log/anvs_evt
DES_.tar-fncheck = /home/data/udpi_log/anvs_des
```

## 性能优化特性

### 1. 并发处理
- 使用goroutine并发检查多个XDR路径
- 内置sync.Mutex确保线程安全
- 支持大规模文件并行处理

### 2. 内存优化
- 流式处理大文件，避免内存溢出
- 智能垃圾回收机制
- 高效的正则表达式引擎

### 3. 错误处理
- 完善的错误恢复机制
- 详细的错误日志记录
- 优雅的异常处理

## 与原Python版本的对比

| 特性 | Python版本 | Go版本 |
|------|------------|--------|
| 执行速度 | 中等 | 快速 |
| 内存使用 | 较高 | 较低 |
| 并发能力 | 有限 | 优秀 |
| 部署复杂度 | 需要Python环境 | 单一二进制文件 |
| 跨平台支持 | 良好 | 优秀 |

## 开发说明

### 依赖管理

项目使用Go Modules进行依赖管理：

```bash
# 添加新依赖
go get github.com/package/name

# 更新依赖
go mod tidy
```

### 测试

```bash
# 运行测试
go test ./...

# 构建测试
go build -o xdr_check_test .
```

### 性能测试

项目包含性能基准测试，可使用以下命令运行：

```bash
go test -bench=. -benchmem
```

## 许可证

本项目基于MIT许可证开源。

## 贡献

欢迎提交Issue和Pull Request来改进项目。