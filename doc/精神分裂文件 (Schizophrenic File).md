**精神分裂文件 (Schizophrenic File)** 是“精神分裂类”在文件系统维度的延伸。在非面向对象编程语言（如 C、Go、JavaScript 模块）中，这种坏味道尤为常见。它打破了模块化的基本准则，将互不相关的业务逻辑、工具函数和数据结构强行堆砌在同一个物理源文件中。

---

# 源码架构分析协议：精神分裂文件 (Schizophrenic File)

## 1. 缺陷定义
**精神分裂文件** 是指一个源文件包含了多组在逻辑上、业务领域上或功能意图上完全孤立的导出实体（函数、变量、常量）。
* **物理特征**：文件长度适中或巨大，但其对外提供的 API 接口缺乏统一的语义中心。
* **内在矛盾**：文件内部形成的多个函数簇之间没有共享状态（如全局变量或私有辅助函数），它们像住在同一个文件夹里的陌生人，彼此没有任何协作。

---

## 2. 典型场景与代码示例

### 场景：演变成“杂物间”的通用模块
开发者习惯于将所有“暂时找不到家”的逻辑塞进 `common` 或 `utils` 文件中，最终导致文件意图分裂。

```c
// 示例：system_bridge.c (典型的分裂文件)
// 现象：文件内部导出的函数服务于三个完全不同的业务领域

#include <stdio.h>
#include <openssl/sha.h>

// --- 人格 A：加密工具簇 (领域：安全) ---
void compute_sha256(const char* data, char* output) {
    // 纯算法逻辑，不依赖文件内其他变量
    SHA256((unsigned char*)data, strlen(data), (unsigned char*)output);
}

// --- 人格 B：数据库连接状态 (领域：存储) ---
static int db_is_connected = 0; // 仅由 B 簇使用
void set_db_status(int status) {
    db_is_connected = status;
}
int get_db_status() {
    return db_is_connected;
}

// --- 人格 C：UI 字符串格式化 (领域：展示) ---
void format_currency_label(double amount, char* buffer) {
    // 纯表现层逻辑
    sprintf(buffer, "$%.2f", amount);
}
```

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别文件级别的“精神分裂”需要量化**函数间的共性引用关系**。

| 全称 (Full Name) | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Functional Disjoint Sets** | **FDS** | **计算方式**：将文件内的导出函数视为节点。若两个函数共同访问了同一个文件级变量，或调用了同一个私有辅助函数，则连一条边。计算其**连通分量**。 | $FDS > 1$ |
| **Domain Diversity Index** | **DDI** | **计算方式**：统计文件导出函数涉及到的外部头文件/包（Imports）的业务领域分布。 | $> 3$ |
| **Inter-cluster Coupling** | **ICC** | **计算方式**：衡量不同函数簇之间相互调用的频率。 | $\approx 0$ |

### FDS (函数不相交集合) 计算过程：
1.  分析函数 $F1, F2$：均使用了文件静态变量 `static_var_A` $\rightarrow$ 簇 1。
2.  分析函数 $F3, F4$：均调用了内部辅助函数 `_private_helper_B` $\rightarrow$ 簇 2。
3.  检查：簇 1 和 簇 2 之间没有任何调用或变量共享。
4.  **结论**：**FDS = 2**。

---

## 4. 缺陷命中规则 (Detection Rules)

判定精神分裂文件的核心规则：

* **规则 1：静态隔离判定**
    $$Rule_{StaticSplit} = (FDS \ge 2) \land (\text{External Domains} \ge 2)$$
    *即：文件内部存在两个以上完全隔离的函数簇，且它们涉及不同的业务领域。*

* **规则 2：工具类离散判定**
    $$Rule_{UtilityScatter} = (FDS > 3) \land (LOC > 500)$$
    *即：文件内存在大量孤立的小函数，没有任何公共依赖，且文件体量已达到维护门槛。*

---

## 5. 检测算法伪代码实现

```python
def detect_schizophrenic_file(file_path):
    ast = parse_file_to_ast(file_path)
    exported_functions = ast.get_exported_functions()
    file_scope_vars = ast.get_static_variables()
    
    # 1. 建立函数连接图
    # 连接条件：共享文件变量 或 调用同一私有函数
    graph = Graph()
    for f1, f2 in combinations(exported_functions, 2):
        if shared_dependencies(f1, f2, file_scope_vars, ast.private_functions):
            graph.add_edge(f1, f2)
            
    # 2. 计算不相交集合
    clusters = graph.get_connected_components()
    fds_count = len(clusters)
    
    # 3. 统计外部域（如通过 include 路径分析）
    domain_count = analyze_import_domains(ast.imports)
    
    if fds_count > 1 and domain_count > 1:
        return True, "Schizophrenic File", clusters
    return False, None
```

---

## 6. 治理建议与详细案例

### 方案 A：物理拆分 (Physical Decoupling) —— 核心方案
**原理**：按照 FDS 检测出的“簇”，将一个文件拆分为多个具有明确命名和职责的文件。
* **重构前**：`system_bridge.c` (包含安全、存储、UI 逻辑)。
* **重构动作**：
    1.  创建 `crypto_utils.c`（移入 SHA256 逻辑）。
    2.  创建 `db_state_manager.c`（移入数据库状态逻辑）。
    3.  创建 `ui_formatter.c`（移入格式化逻辑）。
* **效果**：每个文件实现了真正的“单一职责”，头文件引用更精简，编译依赖更清晰。



### 方案 B：命名空间/包重组
**原理**：在支持命名空间的语言中，至少将这些函数归类到不同的 namespace 或 package 中。
* **案例**：Go 语言中将一个大 package 拆分为多个 sub-package。

---

## 7. 治理决策矩阵

| 表现特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **FDS >= 2 且逻辑属于不同层** | 跨层职责污染（如 UI 混杂 DB） | **最高优先级**：立即按层拆分。 |
| **FDS > 3 但都属于同一业务域** | 模块拆分不够细 | **中优先级**：按子功能模块化拆分。 |
| **存在大量孤立函数且无外部调用** | 死代码或过度封装 | **低优先级**：检查是否是冗余代码，或是该内联到调用处。 |

---
