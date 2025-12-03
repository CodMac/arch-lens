# 🌳 go-treesitter-dependency-analyzer

一个高效的、基于 **Tree-sitter** 的 Go 语言依赖分析工具。本项目专为大规模代码库设计，通过两阶段并发处理流程，精确地从抽象语法树（AST）中提取代码元素（如类、方法、字段）及其相互间的依赖关系。

**当前版本主要支持：** **Java** 语言分析。

## ✨ 核心特性

* **⚡️ 高效解析：** 使用 Tree-sitter 库进行语法解析，速度极快，内存占用低。
* **🧩 两阶段分析：**
    1.  **定义收集 (Phase 1):** 并发解析所有文件，构建全局符号表 (Global Context)。
    2.  **关系提取 (Phase 2):** 利用全局符号表，并发提取跨文件、跨结构的依赖关系。
* **🔗 精确关系类型：** 支持多种依赖关系类型，包括 `CALL`, `IMPORT`, `CREATE`, `EXTEND`, `RETURN` 等。
* **💻 JSON Lines 输出：** 结果以标准的 JSON Lines (JSONL) 格式输出，方便集成到其他数据分析或可视化工具中。

## ⚙️ 架构概览

本项目采用模块化架构，主要分为以下几个核心组件：

| 模块 | 职责 |
| :--- | :--- |
| `cmd/analyzer` | 程序入口点，处理命令行参数和流程调度。 |
| `parser` | 负责文件的读取和 Tree-sitter AST 的生成。 |
| `collector` | (Phase 1) 负责遍历 AST，收集文件内的所有符号定义 (Qualified Names)。 |
| `extractor` | (Phase 2) 负责运行 Tree-sitter Query，利用全局上下文提取依赖关系。 |
| `processor` | 协调并发工作，执行两阶段分析流程。 |
| `model` | 核心数据结构，定义 `CodeElement`, `DependencyRelation`, `GlobalContext` 等。 |
| `x/java` | Java 语言的实现细节，包括语言注册、Collector 和 Extractor 逻辑。 |

## 🚀 快速开始

### 预置条件

由于本项目依赖 `go-tree-sitter`，您需要在环境中安装 **C 编译器 (GCC)**。

* **Linux/macOS:** 通常自带或通过包管理器安装 (`apt install gcc` / `brew install gcc`).
* **Windows:** 推荐安装 **MinGW-w64** 并确保其路径添加到系统环境变量 `%PATH%` 中。

### 1\. 克隆项目

```bash
git clone github.com/CodMac/go-treesitter-dependency-analyzer
cd go-treesitter-dependency-analyzer
```

### 2\. 构建程序

```bash
# 启用 CGO 并构建主程序
CGO_ENABLED=1 go build -o dependency-analyzer ./cmd/analyzer
```

### 3\. 运行分析

运行程序时，您需要指定目标语言和要分析的源文件列表。

```bash
./dependency-analyzer <language> <file1> [file2] [file3]...

# 示例 (分析两个 Java 文件):
./dependency-analyzer java x/java/testdata/User.java x/java/testdata/UserService.java > dependencies.jsonl
```

#### 💡 示例输出 (`dependencies.jsonl`)

分析结果将以 JSON Lines 格式输出到标准输出：

```jsonl
{"Type":"IMPORT","Source":{"Kind":"FILE","Name":"UserService.java",...},"Target":{"Kind":"PACKAGE","Name":"com.example.model.User","QualifiedName":"com.example.model.User"},"Location":{...}}
{"Type":"CREATE","Source":{"Kind":"METHOD","Name":"createNewUser",...},"Target":{"Kind":"CLASS","Name":"User","QualifiedName":"com.example.model.User"},"Location":{...}}
{"Type":"CALL","Source":{"Kind":"METHOD","Name":"processUsers",...},"Target":{"Kind":"METHOD","Name":"println","QualifiedName":"System.out.println"},"Location":{...}}
...
```

## 🧪 运行测试

您可以通过运行测试来验证分析器的功能和准确性。

```bash
# 运行所有测试，包括 parser, collector, extractor, processor
CGO_ENABLED=1 go test ./...
```

## 🛠️ 扩展和贡献

如果您希望为其他语言（如 Go, C++, Python 等）添加支持，您只需要：

1.  在 `go.mod` 中添加相应的 `tree-sitter-<language>` 绑定。
2.  在 `x/` 目录下创建 `x/<language>` 包。
3.  实现 `collector.Collector` 接口和 `extractor.Extractor` 接口的逻辑。
4.  在 `init()` 函数中注册新的语言和组件。

欢迎提交 Pull Requests 来扩展分析功能或修复 Bug！