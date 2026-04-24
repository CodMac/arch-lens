针对 **不稳定依赖（Stable Dependencies Principle, SDP）** 的架构分析方案如下：

---

# 源码架构分析协议：不稳定依赖（SDP）

## 1. 缺陷定义
**不稳定依赖（Stable Dependencies Principle, SDP）** 由 Robert C. Martin 提出，其核心思想是：**“朝着稳定的方向进行依赖”**。
一个模块的**稳定性**取决于它变更的难度。如果一个模块被许多其他模块依赖（高扇入），那么它就是稳定的，因为修改它需要巨大的协调成本。如果一个模块依赖了比自己更不稳定的模块，那么该模块就会被动地因为被依赖项的频繁变动而被迫修改，违反了 SDP 原则。

---

## 2. 典型场景与代码示例

### 2.1 “底层”业务反向依赖“高层”配置 (Common Anti-Pattern)
底层核心逻辑类依赖了经常变动的 UI 逻辑或临时配置文件。
```java
// File: CoreCalculator.java (应该是稳定的核心逻辑)
public class CoreCalculator {
    public double execute() {
        // 错误：核心算法依赖了极易变动的 UI 配置类
        if (MainAppUIConfig.isHighPrecisionMode()) { 
            return 3.1415926;
        }
        return 3.14;
    }
}
```

### 2.2 实体类依赖控制类 (Entity Envies Controller)
数据实体（本应非常稳定）引用了业务流程控制类（经常因需求变动）。
```java
// File: UserEntity.java (数据模型，理应稳定)
public class UserEntity {
    private String name;
    
    // 错误：实体类为了“方便”依赖了复杂的业务处理器
    public void save() {
        RegistrationController.handleSave(this); 
    }
}
```

### 2.3 稳定抽象依赖不稳定实现 (Abstraction Envy Implementation)
定义好的标准接口，在方法签名中使用了具体实现类。
```java
// File: IFileSystem.java (抽象层，理应极其稳定)
public interface IFileSystem {
    // 错误：接口定义中出现了具体实现类 WindowsFileAttributes
    void setAttributes(WindowsFileAttributes attrs); 
}
```

---

## 3. 抽象度量指标 (Metrics)

为了量化 SDP，我们需要计算每个组件的**不稳定性（Instability）**。

| 全称 (Full Name) | 简名 | 描述 | 计算公式 |
| :--- | :--- | :--- | :--- |
| **Afferent Coupling** | **Ca** | **扇入**：外部依赖此组件的类数量。代表**承担的责任**。 | 计数 |
| **Efferent Coupling** | **Ce** | **扇出**：此组件依赖的外部类数量。代表**依赖的压力**。 | 计数 |
| **Instability** | **I** | **不稳定性指标**。范围为 [0, 1]。 | $I = \frac{Ce}{Ca + Ce}$ |

* **$I = 0$**：最高稳定性（大量被依赖，不依赖别人）。
* **$I = 1$**：最高不稳定性（不被依赖，全依赖别人）。

---

## 4. 缺陷命中规则 (Detection Rules)

SDP 违规的核心判定准则：**依赖者的 I 值应该大于被依赖者的 I 值。**

### 规则 1：不稳定性倒置 (SDP Violation)
**定义**: 模块 A 依赖模块 B，但 A 比 B 更稳定。
$$Rule_{SDP} = (A \to B) \land (I(A) < I(B))$$

### 规则 2：核心震荡风险 (Critical)
**定义**: 具有极高扇入（Ca）的稳定模块，依赖了一个极不稳定（I 接近 1）的模块。
$$Rule_{Shock} = (I(A) < 0.2) \land (I(B) > 0.8) \land (A \to B)$$

---

## 5. 检测算法伪代码实现

### 第一阶段：构建组件依赖图并计算 I 值
```python
def calculate_instability(graph):
    instability_map = {}
    for node in graph.nodes:
        ca = get_afferent_count(node) # 谁调我
        ce = get_efferent_count(node) # 我调谁
        if ca + ce == 0:
            i_score = 0
        else:
            i_score = ce / (ca + ce)
        instability_map[node] = i_score
    return instability_map
```

### 第二阶段：扫描违规依赖
```python
def detect_sdp_violations(graph, instability_map):
    violations = []
    for edge in graph.edges:
        source, target = edge.src, edge.dst
        # 如果依赖方的稳定性 > 被依赖方的稳定性，且分值差距超过阈值
        if instability_map[source] < instability_map[target]:
            gap = instability_map[target] - instability_map[source]
            violations.append({
                "from": source, "i_from": instability_map[source],
                "to": target, "i_to": instability_map[target],
                "severity": "HIGH" if gap > 0.5 else "LOW"
            })
    return violations
```

---

## 6. 治理建议与详细案例

### 方案 A：引入依赖倒置（DIP）
**原理**：在不稳定的模块中定义接口，让稳定的模块依赖接口，而非具体实现。

* **案例**：`CoreEngine` (稳定) 直接依赖了 `SmsSender` (不稳定，经常换供应商)。
* **重构**：在 `CoreEngine` 所在包定义 `MessageProvider` 接口。`SmsSender` 实现该接口。
* **结果**：依赖方向反转，`SmsSender` 现在依赖 `CoreEngine` 的接口，符合 SDP。


### 方案 B：剥离不稳定因素
**原理**：如果一个稳定类包含了一小部分不稳定的逻辑，将这部分逻辑提取到专门的不稳定类中。

* **案例**：`UserEntity` (稳定) 内部包含了 `ConfigFetch` (由于网络环境经常变)。
* **重构**：将配置获取逻辑移至 `UserConfigService`。
* **结果**：`UserEntity` 恢复纯净，其 $I$ 值进一步降为 0。

---

## 7. 治理决策矩阵

| 违规严重程度 | 建议方案 | 理由 |
| :--- | :--- | :--- |
| **致命 (Gap > 0.8)** | **依赖倒置 (DIP)** | 核心组件已被污染，必须通过接口物理隔离。 |
| **中等 (Gap 0.4~0.8)** | **功能外放 (Move Method)** | 职责归属错误，将不稳定逻辑移往调用链下游。 |
| **轻微 (Gap < 0.4)** | **持续观察** | 可能是暂时的临时逻辑，若持续震荡则重构。 |

---
