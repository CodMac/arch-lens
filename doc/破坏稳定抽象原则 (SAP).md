**稳定抽象原则 (Stable Abstractions Principle, SAP)** 是对前一项 SDP（稳定依赖原则）的深度补充。它指出：**一个组件的抽象程度应当与其稳定性保持一致。** 简单来说，如果你是一个被大家广泛依赖的核心模块（非常稳定），你必须高度抽象，以便别人可以扩展你而不需要修改你。如果一个模块既稳定又具体，它就会变得“僵化”，难以演进。

---

# 源码架构分析协议：破坏稳定抽象原则 (SAP)

## 1. 缺陷定义
**破坏稳定抽象原则 (SAP)** 是指模块的稳定性与其抽象程度不匹配。
* **稳定模块 (Stable)**：被很多模块依赖（$Ca$ 高），它应该是**抽象**的（包含大量接口/抽象类），以便通过多态应对变化。
* **不稳定模块 (Unstable)**：不被别人依赖（$I$ 接近 1），它应该是**具体**的，直接实现业务逻辑。

违反 SAP 的最危险情况是：**一个模块极其稳定（被全系统依赖），但它是具体的（全是实现逻辑）**。这种情况被称为“痛苦地带”（Zone of Pain）。

---

## 2. 典型场景与代码示例

### 2.1 “僵化的”全局工具类 (Zone of Pain)
一个底层的工具类，被数百个业务类调用，但内部全是静态具体方法，没有接口。
```java
// File: GlobalProcessor.java (极其稳定，Ca 非常大)
public class GlobalProcessor {
    // 缺陷：具体实现被硬编码，由于被广泛依赖，一旦要支持新的处理逻辑，
    // 必须修改这个类，风险极高。
    public void process(String data) {
        if (data.startsWith("A")) { /* ...具体逻辑... */ }
    }
}
```

### 2.2 无意义的过度抽象 (Zone of Uselessness)
一个没有任何依赖项的模块，却定义了极其复杂的接口层次结构。
```java
// File: IInternalHelper.java (完全没有外部依赖)
public interface IInternalHelper {
    void execute();
}
// 这种抽象没有任何人使用，增加了系统的认知负担
public class InternalHelperImpl implements IInternalHelper {
    public void execute() { }
}
```

### 2.3 稳定类引用了非抽象的外部细节
一个处于核心地位的稳定类，其方法参数或返回类型直接使用了具体的第三方实现。
```java
// File: AuthService.java (核心稳定类)
public class AuthService {
    // 缺陷：作为核心接口，返回了具体的第三方库对象，导致所有调用者
    // 都必须耦合这个特定的第三方具体实现。
    public com.thirdparty.SpecificResponse login() { ... }
}
```

---

## 3. 抽象度量指标 (Metrics)

为了量化 SAP，我们需要引入**抽象度 (Abstractness)** 指标。

| 全称 (Full Name) | 简名 | 描述 | 计算公式 |
| :--- | :--- | :--- | :--- |
| **Num of Classes** | **Nc** | 组件内类的总数。 | 计数 |
| **Num of Abstracts** | **Na** | 组件内抽象类和接口的总数。 | 计数 |
| **Abstractness** | **A** | **抽象度指标**。范围 [0, 1]。 | $A = \frac{Na}{Nc}$ |
| **Instability** | **I** | **不稳定性指标**（见 SDP 方案）。 | $I = \frac{Ce}{Ca + Ce}$ |
| **Distance from Main Sequence** | **D** | **偏离主序列距离**。衡量 A 和 I 的匹配度。 | $D = |A + I - 1|$ |

* **$A = 1$**：纯抽象（全是接口）。
* **$A = 0$**：纯具体（全是实现类）。
* **$D = 0$**：完美平衡（模块越稳定则越抽象，越不稳定则越具体）。

---

## 4. 缺陷命中规则 (Detection Rules)

利用 $A$、$I$、$D$ 指标构建规则：

### 规则 1：痛苦地带命中 (Zone of Pain)
**定义**: 模块非常稳定，但几乎没有抽象。这会导致系统极难维护且不可扩展。
$$Rule_{Pain} = (I < 0.2) \land (A < 0.2)$$

### 规则 2：无用抽象命中 (Zone of Uselessness)
**定义**: 模块完全不稳定（没人用），却做了大量的抽象。这通常是过度工程。
$$Rule_{Useless} = (I > 0.8) \land (A > 0.8)$$

### 规则 3：严重偏离主序列 (D-Score Violation)
**定义**: 模块的稳定性和抽象度严重失衡。
$$Rule_{D\_Violation} = D > 0.5$$

---

## 5. 检测算法伪代码实现

### 第一阶段：计算 A-I 指标对
```python
def analyze_sap_metrics(components):
    results = []
    for comp in components:
        i_score = calculate_instability(comp) # 获取 SDP 中的 I
        a_score = calculate_abstractness(comp) # Na / Nc
        
        # 计算偏离距离 D
        d_score = abs(a_score + i_score - 1)
        
        results.append({
            "name": comp.name,
            "A": a_score,
            "I": i_score,
            "D": d_score
        })
    return results
```

### 第二阶段：可视化分布与规则判定
将结果映射到 A-I 坐标轴（主序列图）。


```python
def check_sap_rules(results):
    for r in results:
        if r['A'] < 0.2 and r['I'] < 0.2:
            trigger_warning("Zone of Pain: 建议提取接口或引入多态")
        elif r['A'] > 0.8 and r['I'] > 0.8:
            trigger_warning("Zone of Uselessness: 建议合并抽象，消除过度设计")
        elif r['D'] > 0.5:
            trigger_warning("Distancing Violation: 稳定性与抽象度不匹配")
```

---

## 6. 治理建议与详细案例

### 方案 A：提取接口（针对痛苦地带）
**原理**：为高度稳定的具体类创建抽象接口，将业务逻辑移至实现类中。

* **案例**：`FileSystemStorage` 被全系统调用。
* **重构**：定义 `IStorage` 接口，让 `FileSystemStorage` 成为其中一种实现。
* **结果**：其他模块依赖 `IStorage` ($A=1$)，稳定性保持不变，但扩展性提升。

### 方案 B：依赖倒置配合插件化
**原理**：核心模块只保留抽象，具体逻辑通过插件或策略模式（Strategy Pattern）注入。

* **案例**：`ReportEngine` 包含各种具体格式（PDF, Excel）的解析代码。
* **重构**：`ReportEngine` 只定义 `ReportFormatter` 接口。
* **结果**：`ReportEngine` 变得极其稳定且抽象 ($I \to 0, A \to 1$)，新格式只需新增具体类。

---

## 7. 治理决策矩阵

| 坐标象限 | 现状描述 | 治理动作 |
| :--- | :--- | :--- |
| **(I low, A low)** | **痛苦地带** | **最高优先级**。必须抽象化，否则系统将陷入僵化。 |
| **(I high, A high)** | **无用抽象** | **低优先级**。清理冗余接口，简化代码。 |
| **(I high, A low)** | **正常不稳定区** | 无需动作。这是业务代码（如 Controller）应有的状态。 |
| **(I low, A high)** | **正常稳定抽象区** | 无需动作。这是框架/核心库（如 JDBC 接口）应有的状态。 |

---
