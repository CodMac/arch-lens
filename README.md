# 🌳 Go Tree-sitter Dependency Analyzer

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Tree-sitter](https://img.shields.io/badge/Parser-Tree--sitter-green)](https://tree-sitter.github.io/)

**Go Tree-sitter Dependency Analyzer** 是一个高性能的代码依赖分析工具。它专为大规模代码库设计，利用 **Tree-sitter** 的增量解析能力和 Go 的并发特性，精确地从源码中提取代码元素（类、方法、字段）及其相互依赖关系。

该工具采用 **两阶段分析（Two-Phase Analysis）** 策略，能够有效解决跨文件符号解析问题，生成包含丰富元数据的结构化依赖图。

## ✨ 核心特性

*   **⚡️ 高性能并发架构**：基于 Go协程（Goroutines）的 Worker Pool 设计，支持并发解析和提取，充分利用多核 CPU。
*   **🧩 两阶段精确分析**：
*   **Phase 1 (Definition)**：构建全局符号表（Global Context），记录全限定名（Qualified Name）、包结构和导入关系。
*   **Phase 2 (Extraction)**：基于符号表解析引用，识别跨文件的复杂依赖。
*   **☕️ 深度 Java 支持**（当前重点）：
*   支持 **类 (Class)**、**接口 (Interface)**、**枚举 (Enum)** 及 **注解 (Annotation)**。
*   支持 **内部类** 和 **内部枚举** 的嵌套结构分析。
*   精确提取 **修饰符** (public, static, final)、**泛型签名**、**异常抛出** 等元数据。
*   **🔗 丰富的关系类型**：
*   `CALL` (方法调用), `IMPORT` (导入), `create` (对象创建), `EXTEND` (继承), `IMPLEMENT` (实现)。
*   `USE` (字段/变量使用), `CAST` (类型强转), `THROW` (异常抛出), `ANNOTATION` (注解修饰), `CONTAIN` (结构包含)。
*   **🛠 AST 可视化调试**：支持将源文件的 AST 导出为格式化的 S-expression (`.ast.format`)，便于调试解析逻辑。

## ⚙️ 项目结构

```text
.
├── collector/       # Phase 1: 定义收集器接口与工厂
├── extractor/       # Phase 2: 关系提取器接口与工厂
├── model/           # 核心数据模型 (CodeElement, DependencyRelation, GlobalContext)
├── parser/          # Tree-sitter 解析器封装，AST 生成
├── processor/       # 流程控制器，调度并发 Worker 执行两阶段分析
├── output/          # 结果输出处理 (JSON Lines)
├── x/               # 语言特定实现扩展包
│   └── java/        # Java 语言的 Collector 和 Extractor 实现
├── main.go          # 程序入口，命令行参数处理
└── go.mod           # 依赖定义
```

## 🚀 快速开始

### 1. 环境准备

由于依赖 `go-tree-sitter`，构建环境需要安装 **C 编译器**。

*   **Linux/macOS**: GCC (通常默认安装)
*   **Windows**: 推荐安装 [MinGW-w64](https://www.mingw-w64.org/) 并配置 PATH。

### 2. 构建项目

```bash
git clone https://github.com/CodMac/go-treesitter-dependency-analyzer.git
cd go-treesitter-dependency-analyzer

# 启用 CGO 编译 (必须)
CGO_ENABLED=1 go build -o dependency-analyzer main.go
```

### 3. 运行分析

**基本用法**:
```bash
./dependency-analyzer -lang <language> -path <source_path> [options]
```

**示例**: 分析当前目录下的 Java 项目，输出到文件：

```bash
./dependency-analyzer -lang java -path ./src -jobs 8 > output.jsonl
```

**常用参数**:

| 参数 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `-lang` | `go` | 目标分析语言 (目前支持完整特性的为 `java`) |
| `-path` | `.` | 源代码目录或文件路径 |
| `-filter` | `""` | 文件名过滤正则表达式 (例如: `".*\.java$"`) |
| `-jobs` | `4` | 并发 Worker 数量 |
| `-output-ast` | `false` | 是否输出解析后的 AST 文件 (`.ast`) 用于调试 |
| `-format-ast` | `true` | 是否格式化输出的 AST 文件 |

## 📄 输出格式

结果以 **JSON Lines (JSONL)** 格式输出，每行代表一个依赖关系。

### JSON 示例

```json
{
  "Type": "CALL",
  "Source": {
    "Kind": "METHOD",
    "Name": "findById",
    "QualifiedName": "com.example.service.UserService.findById",
    "Path": "src/com/example/service/UserService.java",
    "Signature": "public User (String id)",
    "Extra": {
      "ReturnType": "User",
      "MethodExtra": { "Parameters": ["String id"] }
    }
  },
  "Target": {
    "Kind": "METHOD",
    "Name": "findOne",
    "QualifiedName": "com.example.service.UserRepository.findOne"
  },
  "Location": {
    "FilePath": "src/com/example/service/UserService.java",
    "StartLine": 25,
    "EndLine": 25,
    "StartColumn": 20,
    "EndColumn": 42
  }
}
```

### 关键字段说明

*   **`Type`**: 依赖类型 (如 `CALL`, `IMPORT`, `EXTEND` 等)。
*   **`Source` / `Target`**:
*   `Kind`: 元素类型 (`CLASS`, `METHOD`, `FIELD`, `INTERFACE`, `ENUM` 等)。
*   `QualifiedName`: 全限定名（例如 `com.pkg.Class.method`），用于唯一标识符号。
*   `Extra`: 包含语言特定的详细信息，如 Java 的修饰符 (`public static`)、注解列表、父类、接口实现列表等。

## 🛠️ 扩展新语言

项目采用插件化架构，添加新语言支持（如 Python 或 Go）非常简单：

1.  在 `x/` 目录下创建新语言包 (例如 `x/python`)。
2.  实现 `collector.Collector` 接口：定义如何从 AST 中收集符号定义。
3.  实现 `extractor.Extractor` 接口：编写 Tree-sitter Queries 提取依赖关系。
4.  在 `init()` 函数中调用 `parser.RegisterLanguage` 等方法注册组件。
5.  在 `main.go` 中导入该包：`_ "github.com/.../x/python"`。

## 🧪 测试

项目包含完整的单元测试，覆盖 Parser、Collector 和 Processor 逻辑。

```bash
# 运行所有测试
CGO_ENABLED=1 go test ./...

# 运行特定测试（如 Java 部分）
CGO_ENABLED=1 go test ./x/java/... -v
```

## 📜 License

MIT License