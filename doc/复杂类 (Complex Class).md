这是一份集成度最高、包含了 **Response For a Class (RFC)** 修复逻辑及详细计算示例的《复杂类 (Complex Class) 检测与分析方案》。你可以直接复制使用。

---

# 源码架构分析协议：复杂类 (Complex Class)

## 1. 缺陷定义
**复杂类 (Complex Class)**，常被称为 **Blob Class** 或 **Brain Class**。它与“上帝类”不同，不一定管辖范围极广，但其**内部逻辑深度**和**维护风险**极高。这类类通常专注于某一核心业务，但由于内部充斥着深层嵌套、密集的条件分支以及庞大的外部调用链，导致代码极其晦涩，成为系统中最脆弱、最不敢触碰的“黑盒”。

---

## 2. 典型场景与代码示例

### 2.1 嵌套分支黑洞 (Nested Conditional Hell)
逻辑深度过大，开发者必须在大脑中维护极其复杂的堆栈才能理解当前分支。
```java
public class TaxCalculator {
    public double calculate(User user, Order order) {
        // 缺陷：逻辑深度过大，认知成本极高
        if (user.isForeign()) {
            if (order.getType() == EXPORT) {
                if (isVatFree(user)) {
                    // ... 嵌套 5 层 ...
                } 
            }
        } else {
            // ... 又是 5 层分支 ...
        }
        return result;
    }
}
```

### 2.2 响应范围过广的“章鱼类” (The Octopus Class)
类本身代码不多，但它像章鱼一样伸出触角，调用了全系统数十个不同类的方法，任何外部变动都会波及它。

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

复杂类的检测核心在于评估**单个方法的逻辑深度**与**整体响应范围**。

| 全称 | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Max Cyclomatic Complexity** | **MCC** | **计算方式**：类中单个方法的最大圈复杂度。统计逻辑流图中判定节点的数量。 | $> 10$ |
| **Max Cognitive Complexity** | **CogC** | **计算方式**：基于嵌套深度的加权复杂度。每增加一层 `if/for`，权重递增。 | $> 15$ |
| **Weighted Methods per Class** | **WMC** | **计算方式**：$WMC = \sum CC_i$。全类所有方法圈复杂度之和。 | $> 50$ |
| **Response For a Class** | **RFC** | **计算方式**：$RFC = \|M \cup R\|$。见下文详细详解。 | $> 50$ |

### 重点指标详解：Response For a Class (RFC)
**定义**：一个类响应集合的大小。它衡量该类执行时潜在可能触达的方法总数。
**公式**：$RFC = \|M \cup R\|$
* **$M$**: 本类定义的所有方法集合。
* **$R$**: 本类方法**直接调用**的外部方法去重集合。

**计算示例：**
```java
public class OrderService {
    public void create() {
        validate();      // 内部方法 M2
        db.save();       // 外部方法 R1
        log.info();      // 外部方法 R2
    }
    private void validate() {
        checkData();     // 外部方法 R3
    }
}
// M = {create, validate} (2)
// R = {db.save, log.info, checkData} (3)
// RFC = 2 + 3 = 5
```

**计算规则：**

  1. 去重：如果多个内部方法调用了同一个外部方法，该外部方法只计数一次。
  2. 多态：如果调用的是接口或抽象方法，通常按声明的方法计数，不展开计算所有子类实现。
  3. 继承：通常只计算当前类定义的逻辑，除非该类重写了父类方法并显式调用了 super。

### 重点指标详解：认知复杂度 (Cognitive Complexity)
```
void check(int x) {
    if (x > 0) {       // +1
        if (x < 10) {  // +2 (嵌套得分 1 + 基础得分 1)
            System.out.println("Low");
        }
    }
}
// 圈复杂度为 2，但认知复杂度为 3。因为它考虑了理解嵌套带来的心理压力。
```

---

## 4. 缺陷命中规则 (Detection Rules)

判定复杂类的核心在于：**局部逻辑晦涩** 或 **副作用范围失控**。

### 规则 1：关键逻辑过载 (Critical Logic Overload)
只要类中有一个方法的认知复杂度超过人类理解极限。
$$Rule_{SingleMethod} = (MCC > 15) \lor (CogC > 20)$$

### 规则 2：响应集超标 (High Response Risk)
类整体复杂度较高，且其响应集合（RFC）过大，导致维护风险失控。
$$Rule_{HighRisk} = (WMC > 50) \land (RFC > 60)$$

---

## 5. 检测算法伪代码实现

```python
def detect_complex_class(class_nodes):
    for cls in class_nodes:
        # 1. 扫描圈复杂度与认知复杂度
        max_mcc = max(calculate_cc(m) for m in cls.methods)
        max_cogc = max(calculate_cog(m) for m in cls.methods)
        wmc = sum(calculate_cc(m) for m in cls.methods)
        
        # 2. 计算 RFC (响应集合)
        internal_methods = set(cls.methods)
        external_calls = set()
        for m in cls.methods:
            external_calls.update(m.get_direct_external_calls())
        rfc = len(internal_methods | external_calls)
        
        # 3. 规则匹配
        if max_mcc > 15 or max_cogc > 20:
            report_issue(cls, "Complex Method detected", {"MCC": max_mcc, "CogC": max_cogc})
        elif wmc > 50 and rfc > 60:
            report_issue(cls, "High Risk Complex Class", {"WMC": wmc, "RFC": rfc})
```

---

## 6. 治理建议与详细案例

### 方案 A：卫语句重构 (Guard Clauses) —— 降低 CogC
**原理**：将嵌套的 `if-else` 转换为平级的卫语句，消除嵌套带来的认知压力。
* **案例**：`TaxCalculator` 中的 5 层嵌套。
* **重构**：使用 `if (condition) return;` 提前结束分支。
* **结果**：`CogC` 从 20+ 降至 5 左右。

### 方案 B：提取方法 (Extract Method) —— 降低单方法 MCC
**原理**：将巨型方法按功能块拆分为多个语义清晰的小方法。
* **案例**：一个 300 行的 `process()` 方法。
* **重构**：拆分为 `validateData()`、`applyDiscount()`、`saveResult()`。

### 方案 C：引入中介者 (Mediator) —— 降低 RFC
**原理**：如果类调用的三方服务太多，通过一个中介者封装这些交互。
* **结果**：该类的 `RFC` 集合中，外部方法 $R$ 的数量将显著减少。

---

## 7. 治理决策矩阵

| 指标表现 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **CogC 极高** | 逻辑嵌套太深 | **最高优先级**：执行“卫语句重构”或“状态模式”。 |
| **RFC > 80** | 副作用范围不可控 | **中优先级**：检查“迪米特法则”，减少跨类调用链。 |
| **WMC > 100** | 类整体过度肥胖 | **中优先级**：结合“上帝类”治理方案进行职责拆分。 |

---

**接下来，我们是否继续进行第 10 项：功能依恋 (Feature Envy) 的分析？**