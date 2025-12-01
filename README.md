# 🔗 Go-TreeSitter-Dependency-Analyzer

一个高性能、多语言的源代码依赖关系分析工具。本项目利用 **Tree-sitter** 的强大能力解析 AST，通过**两阶段并发处理**和**全局符号表**机制，精确提取项目中的各种依赖关系，并以 JSON Lines (JSONL) 格式输出。

## ✨ 核心特性

* **多语言支持**: 内建支持 Java 和 Go，并易于扩展到其他 Tree-sitter 支持的语言。
* **两阶段解析**: 采用 `Definition Pass` (收集定义) 和 `Relation Pass` (提取关系) 两阶段处理，实现准确的跨文件限定名 (Qualified Name, QN) 解析。
* **全局符号表**: 通过 `GlobalContext` 聚合项目所有定义，解决了传统静态分析中复杂的跨文件引用问题。
* **并发高效**: 使用 Go 协程 (`processor`) 实现文件解析和关系提取的并发调度，充分利用多核资源。
* **标准化输出**: 所有依赖关系均输出为易于处理的 JSON Lines (JSONL) 格式。

## ⚙️ 技术栈

* **核心语言**: Go (Golang)
* **AST 解析**: Tree-sitter (通过 `github.com/smacker/tree-sitter` Go 绑定)
* **语言支持**:
    * Java: `github.com/tree-sitter/tree-sitter-java`
    * Go: `github.com/smacker/tree-sitter-go`
* **数据模型**: 自定义 `DependencyRelation` 模型。

## 🚀 快速开始

### 1. 安装依赖

确保您已安装 Go 1.18+ 环境。在项目根目录下，运行以下命令拉取所有 Go 依赖和 C 编译的 Tree-sitter 绑定：

```bash
go mod tidy
````

### 2\. 构建项目

```bash
go build -o dependency-analyzer
```

### 3\. 运行分析

使用 `-lang` 指定语言，`-path` 指定目标文件或目录。

**分析 Java 项目:**

```bash
./dependency-analyzer -lang java -path ./path/to/java/project > dependencies.jsonl
```

**分析 Go 项目 (需要完善提取器):**

```bash
./dependency-analyzer -lang go -path ./path/to/go/project > dependencies.jsonl
```

### 命令行参数

| 参数名 | 默认值 | 描述 |
| :--- | :--- | :--- |
| `-path` | `.` | 要分析的源代码目录或单个文件路径。 |
| `-lang` | `java` | 要分析的编程语言 (`java`, `go` 等)。 |
| `-workers` | `runtime.NumCPU()` | 并发处理文件的协程数量。 |

## 📐 依赖关系模型

项目的核心输出是 `DependencyRelation` 结构体，以 JSON Lines (JSONL) 格式输出到标准输出。

### JSONL 输出示例

```jsonl
{"Type":"IMPORT","Source":{"Kind":"FILE","Name":"...","QualifiedName":"...","Path":"src/Main.java"},"Target":{"Kind":"PACKAGE","Name":"java.util.List","QualifiedName":"java.util.List"},"Location":{"FilePath":"src/Main.java","StartLine":3,"EndLine":3,"StartColumn":1,"EndColumn":20}}
{"Type":"CALL","Source":{"Kind":"METHOD","Name":"main","QualifiedName":"com.example.Main.main","Path":"src/Main.java"},"Target":{"Kind":"METHOD","Name":"println","QualifiedName":"java.io.PrintStream.println"},"Location":{"FilePath":"src/Main.java","StartLine":10,"EndLine":10,"StartColumn":8,"EndColumn":25}}
{"Type":"CONTAIN","Source":{"Kind":"CLASS","Name":"Main","QualifiedName":"com.example.Main","Path":"src/Main.java"},"Target":{"Kind":"METHOD","Name":"main","QualifiedName":"com.example.Main.main"},"Location":{"FilePath":"src/Main.java","StartLine":8,"EndLine":8,"StartColumn":5,"EndColumn":20}}
```

### 字段说明

| 字段 | 类型 | 描述 |
| :--- | :--- | :--- |
| `Type` | string | 依赖类型 (`CALL`, `IMPORT`, `EXTEND`, `USE`, `CONTAIN`, `PARAMETER`, `RETURN` 等)。 |
| `Source` | CodeElement | 关系的源头实体（调用者、导入者等）。 |
| `Target` | CodeElement | 关系的目标实体（被调用函数、被导入包等）。 |
| `Location` | Location | 关系在源码中发生的位置。 |

## 🏗️ 架构概览

本项目采用清晰的分层架构：

1.  **`model/`**: 数据模型和全局符号表(`GlobalContext`)定义。
2.  **`parser/`**: Tree-sitter 基础封装层，负责加载语言和解析文件。
3.  **`extractor/`**: **语言适配层**，包含 `DefinitionCollector` 和 `ContextExtractor` 接口的实现。
      * `extractor/java/`: Java 语言的提取实现。
      * `extractor/golang/`: Go 语言的提取实现 (待完善)。
4.  **`processor/`**: 并发调度层，负责管理文件队列和执行两阶段分析。
5.  **`output/`**: 结果格式化层，负责将模型结构体写入 JSON Lines。

## 🧩 扩展新语言

要添加新的语言支持，您只需要：

1.  引入新的 Tree-sitter 绑定库（例如 `tree-sitter-python`）。
2.  在 `extractor/` 下创建新的语言目录（例如 `extractor/python/`）。
3.  实现 `extractor.Extractor` 接口，包括 `CollectDefinitions` 和 `Extract` 两个阶段的逻辑。
4.  在 `main.go` 中导入新的语言包以触发注册。

-----

> **注意：** 尽管代码框架已完成，但 Go 语言和 Java 提取器中的 QN 解析逻辑 (`resolveQualifiedName`) 仍然依赖于项目级别的上下文信息。对于复杂的动态引用，可能需要集成更高级的类型推导机制。

```
```