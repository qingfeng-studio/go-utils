# go-utils

[![Go Report Card](https://goreportcard.com/badge/github.com/yourname/go-utils)](https://goreportcard.com/report/github.com/yourname/go-utils)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/yourname/go-utils.svg)](https://pkg.go.dev/github.com/yourname/go-utils)

`go-utils` 是一个 Go 语言的通用工具库集合，旨在为 Go 开发者提供一系列开箱即用、经过验证的常用功能模块。它封装了日志、HTTP 客户端、数据处理、加密解密等高频需求，帮助你快速搭建项目，减少重复造轮子，专注于核心业务逻辑。

---

## 📦 目录结构与功能简介

`go-utils` 采用模块化设计，每个功能模块独立成包，你可以根据需要选择性导入。

| 目录/包名 | 作用 |
| :--- | :--- |
| **`utils/`** | **核心工具包**。提供最基础、最广泛使用的通用函数，如空值判断、错误处理简化、环境变量读取等。是整个库的“门面”之一。 |
| **`logger/`** | **日志封装**。基于 `zap` 日志库进行封装，提供简洁的初始化接口、结构化日志输出和日志级别控制。让你在项目中快速集成高性能日志。 |
| **`httpx/`** | **增强 HTTP 客户端**。提供一个功能丰富的 HTTP 客户端，内置超时控制、自动重试机制（可配置），并预留了中间件扩展点（如日志、熔断），简化对外部 API 的调用。 |
| **`sugar/`** | **数据类型“语法糖”**。提供对字符串 (`string`)、切片 (`slice`)、映射 (`map`) 等内置数据类型的便捷操作函数，如 `Join`, `Reverse`, `Map`, `Filter`, `Merge` 等，让代码更简洁易读。 |
| **`crypto/ace/`** | **ACE 加解密**。提供基于特定算法（此处指代你的 `ace` 实现）的加解密功能。包含加密、解密、密钥管理等接口，用于保护敏感数据。 |
| **`config/`** | **配置加载**。支持从 YAML 或 JSON 配置文件中加载配置，并能与环境变量结合使用（环境变量优先级更高），方便在不同环境（开发、测试、生产）下管理应用配置。 |
| **`internal/`** | **内部实现**。存放各模块共享的、不对外暴露的底层实现细节。此目录下的代码仅供 `go-utils` 内部使用。 |

---

## 🚀 快速开始

### 1. 安装

```bash
go get github.com/yourname/go-utils
```

## 🧪 测试命令速查表

### 基础测试

```bash
# 进入logger目录
cd logger

# 运行所有测试
go test -v

# 运行测试并显示覆盖率
go test -v -cover

# 运行特定测试
go test -run=TestConcurrentAccess -v

# 运行单元测试并显示详细信息
go test -v -run=TestLoggerInfo -json

# 根目录直接执行某个目录下的测试用例
go test ./logger -v
```

### 基准测试

```bash
# 运行所有基准测试
go test -run=^$ -bench=. -benchmem

# 运行特定基准测试
go test -run=^$ -bench=BenchmarkLoggerInfo -benchmem

# 多次运行基准测试
go test -run=^$ -bench=. -benchmem -count=5
```

### 覆盖率测试

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
```

### 性能分析

```bash
# CPU性能分析
go test -run=^$ -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# 内存分析
go test -run=^$ -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

### 竞态检测

```bash
# 运行竞态检测
go test -race -v

# 基准测试 + 竞态检测
go test -run=^$ -bench=. -race
```

### 一键测试脚本

创建 `test.sh`:

```bash
#!/bin/bash
echo "=== 运行所有测试 ==="
go test -v

echo -e "\n=== 基准测试 ==="
go test -run=^$ -bench=. -benchmem

echo -e "\n=== 覆盖率报告 ==="
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

echo -e "\n=== 竞态检测 ==="
go test -race -v

echo "测试完成！"
```

使用方法：
```bash
chmod +x test.sh
./test.sh
```