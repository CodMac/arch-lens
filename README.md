# Arch-Lens

**Arch-Lens** 是一款专为大规模复杂系统设计的**高精度静态架构分析引擎**。它通过深度感知代码语义，能够还原包含泛型细节、闭包捕获、流式调用在内的深层依赖关系，并直接生成可交互的架构拓扑图。

---

## 📊 核心数据模型 (Dependency Relations)

Arch-Lens 能够识别并提取以下维度的依赖关系，通过 `Mores` 字典提供极细粒度的元数据分析支持：

| 关系类型 (Type) | 目标种类 (Target Kind) | 说明 | 核心元数据 (Mores) 举例 |
| --- | --- | --- | --- |
| **Contain** | Package, File, Element | 拓扑包含关系（包、文件、成员） | - |
| **Import** | File, External | 源码级别的导入依赖 | `raw_import_path` |
| **Extend** | Class, Interface | 类/接口继承 | `is_inherited` |
| **Implement** | Interface | 接口实现 | - |
| **Call** | Method | 方法调用、构造函数、方法引用 | `is_chained`, `is_functional` |
| **Create** | Class | 对象实例化、数组创建 | `is_array`, `variable_name` |
| **Assign** | Variable, Field | 变量赋值、复合赋值、自增减 | `operator`, `is_initializer` |
| **Use** | Variable, Field | 标识符引用、字段访问 | **`is_capture`**, `usage_role` |
| **Capture** | Variable, Field | **闭包捕获**：Lambda 引用外部变量 | `capture_depth`, `is_effectively_final` |
| **TypeArg** | Class | 泛型参数依赖 | `type_arg_index`, `type_arg_depth` |
| **Parameter** | Class | 方法形参类型依赖 | `parameter_index`, `is_varargs` |
| **Return** | Class | 方法返回值类型依赖 | `is_primitive`, `is_array` |
| **Throw** | Class | 异常抛出（声明或主动抛出） | `is_runtime`, `is_rethrow` |
| **Annotation** | KAnnotation | 注解引用 | `annotation_target` |
| **Cast** | Class | 类型转换、Instanceof 检查 | `is_pattern_matching` |

---

## 🏗 核心架构

Arch-Lens 将分析逻辑抽象为五个标准阶段，支持高并发流水线作业：

1. **Collector**：提取原始定义与元数据。
2. **Resolver**：执行符号绑定，处理 Import 与通配符。
3. **Extractor**：执行 Action Query，发现动态行为依赖。
4. **Linker**：缝合全局拓扑网，构建层级结构。
5. **NoiseFilter**：执行降噪策略（Raw/Balanced/Pure）。

---

## 🛣 路线图 (Roadmap)

### 核心引擎 (Core Engine)

* [ ] **SQL/Cypher Adapter**：支持将结果导入 Neo4j 数据库。
* [ ] **Diff Analysis**：对比两次 Commit 间的架构耦合变化。
* [ ] **Increment Mode**：基于 Git 修改范围的增量解析。

### 语言支持 (Language Support)

* **Java (现已支持)**:
* [ ] **语法糖增强**：自动生成 `Enum.values()`、`Record` Getter 等隐式成员。
* [ ] **框架增强**：支持 Lombok (`@Data`, `@Builder`) 等编译时生成方法的语法糖注入。
* [ ] **推断增强**：Lambda 目标函数式接口的 Expected Type 自动推断。


* **Go (开发中)**:
* [ ] Interface 隐式实现映射。
* [ ] Struct Embedding 关系识别。



---

## 🛠 快速开始

### 编译 (Build)

Arch-Lens 依赖 `tree-sitter` 动态库，编译前请确保环境已配置相关依赖：

```bash
# 克隆项目
git clone https://github.com/CodMac/arch-lens.git
cd arch-lens

# 下载依赖
go mod download

# 编译 CLI 工具
go build -o arch-lens cmd/main.go

```

### 基础使用 (Usage)

Arch-Lens 提供了强大的命令行接口，支持多种输出格式和过滤等级。

#### 1. 基础 JSONL 导出

分析指定目录，并将结果导出为可供后续处理的 `jsonl` 格式：

```bash
./arch-lens -path ./my-project -lang java -level 1 -format jsonl -out-dir ./result

```

#### 2. 生成 Mermaid 可视化图谱

如果项目规模适中（默认限制 200 节点内），可以直接导出 HTML 格式的 Mermaid 拓扑图：

```bash
./arch-lens -path ./my-project -format mermaid -out-dir ./visual

```

#### 3. 核心参数说明

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-lang` | `java` | 分析的目标语言 |
| `-path` | `.` | 源码根路径 |
| `-level` | `1` | **过滤等级**：0(Raw), 1(Balanced), 2(Pure) |
| `-format` | `jsonl` | **输出格式**：jsonl 或 mermaid |
| `-jobs` | `4` | 并发执行的任务数 |
| `-filter` | `""` | 文件过滤正则表达式 |

---

## 🤝 贡献与许可

* 欢迎提交 PR 增加新语言支持（如 Go, C++, Python）。
* 如果你发现了特定的 Java 语法糖未能解析，请提交 Issue 并附带代码片段。
* 本项目基于 **MIT License** 开源。