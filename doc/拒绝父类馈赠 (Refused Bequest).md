**拒绝父类馈赠 (Refused Bequest)** 是一种典型的面向对象设计“坏味道”，最早由 Martin Fowler 在《重构》中定义。它描述了继承关系中的一种病态：子类继承了父类，但却通过各种手段（如抛出异常、空实现）来“拒绝”父类提供的功能。

---

# 源码架构分析协议：拒绝父类馈赠 (Refused Bequest)

## 1. 缺陷定义
当子类继承自一个父类，却不使用父类提供的属性或方法时，就发生了 **拒绝父类馈赠**。这通常意味着继承体系违背了逻辑上的“Is-A”关系。开发者往往只是为了复用父类的某一个微小功能而强行继承，导致子类携带了大量不属于它的“行李”。

虽然这种行为不一定会直接导致崩溃，但它会导致代码的可读性极差，并严重误导调用者对多态行为的预期。

---

## 2. 典型场景与代码示例

### 2.1 抛出异常的“假性重写” (The Exception Thrower)
子类为了“关闭”父类的某个功能，在重写方法时抛出不受支持的异常。
```java
// 父类：通用打印机
public class Printer {
    public void print() { /* 打印逻辑 */ }
    public void scan()  { /* 扫描逻辑 */ }
}

// 子类：低端打印机（拒绝扫描功能）
public class BudgetPrinter extends Printer {
    @Override
    public void scan() {
        // 缺陷：拒绝了父类的“馈赠”，破坏了调用者的预期
        throw new UnsupportedOperationException("该机型不支持扫描");
    }
}
```

### 2.2 空实现或“静默拒绝” (The Silent Refuser)
子类重写了方法但内部没有任何逻辑，或者直接返回 `null/false`。
```java
// 场景：为了复用“移动”逻辑而继承了“鸟类”
public class Bird {
    public void fly() { /* 复杂的飞行算法 */ }
}

public class Ostrich extends Bird {
    @Override
    public void fly() {
        // 鸵鸟不会飞，所以这里变为空实现
        // 缺陷：如果一段代码遍历 Bird 列表让它们 fly，鸵鸟会静默失效
    }
}
```

### 2.3 仅为了局部代码复用而继承 (Utility Inheritance)
子类仅仅想要父类的一个工具方法，却继承了父类所有的状态，导致子类内部出现大量不相关的成员变量。

---

## 3. 抽象度量指标 (Metrics)

为了量化子类对父类的利用率，我们需要定义以下指标：

| 全称 (Full Name) | 简名 | 描述 | 计算公式 |
| :--- | :--- | :--- | :--- |
| **Bequest Utilization** | **BU** | **馈赠利用率**。子类真正使用的父类方法/属性比例。 | $BU = \frac{U_{sub}}{P_{total}}$ |
| **Overridden Void Ratio** | **OVR** | **无效重写率**。重写方法中包含抛异常或空实现的比例。 | $OVR = \frac{N_{ignored}}{N_{overridden}}$ |
| **Inherited Depth** | **ID** | **继承深度**。该类在继承树中的深度。 | 计数 |

* **$U_{sub}$**: 子类（及其实例调用者）实际调用的父类公开方法数。
* **$P_{total}$**: 父类定义的公开方法总数。

---

## 4. 缺陷命中规则 (Detection Rules)

### 规则 1：强力拒绝规则 (Hard Refusal)
**定义**: 子类显式地通过抛出特定异常来拒绝父类方法。
$$Rule_{Hard} = (OVR > 0) \land (\text{Exception} \in \{UnsupportedOperation, NotImplemented\})$$

### 规则 2：低利用率继承 (Weak Utilization)
**定义**: 子类继承了大量方法，但实际使用的比例极低。
$$Rule_{Lazy} = (BU < 0.2) \land (P_{total} > 5)$$

### 规则 3：逻辑违背 (Logic Violation)
**定义**: 结合 LSP（里氏替换原则），如果子类在任何场景下无法替换父类，即视为严重拒绝。

---

## 5. 检测算法伪代码实现

### 第一阶段：扫描重写方法体
```python
def detect_refusal_patterns(class_node):
    ignored_count = 0
    overridden_methods = class_node.get_overridden_methods()
    
    for method in overridden_methods:
        body = method.get_body()
        # 模式匹配：空实现或直接抛异常
        if body.is_empty() or body.only_throws_unsupported():
            ignored_count += 1
            
    return ignored_count / len(overridden_methods) if overridden_methods else 0
```

### 第二阶段：计算利用率
```python
def calculate_bu(sub_class, parent_class):
    parent_methods = parent_class.get_public_methods()
    used_count = 0
    
    # 扫描子类代码，看是否调用了 super.xxx() 
    # 或外部调用者是否通过子类实例使用了父类方法
    for m in parent_methods:
        if is_method_ever_used_in_sub(m, sub_class):
            used_count += 1
            
    return used_count / len(parent_methods)
```

---

## 6. 治理建议与详细案例

### 方案 A：组合优于继承 (Favor Composition over Inheritance)
**原理**：如果子类只需要父类的部分功能，不应继承，而应将父类作为子类的一个**成员属性**。

* **案例**：上面的 `BudgetPrinter`。
* **重构**：创建一个 `PrintEngine` 类。让 `StandardPrinter` 和 `BudgetPrinter` 都持有它。
* **结果**：`BudgetPrinter` 不再拥有 `scan()` 方法，也就没有拒绝的问题。


### 方案 B：提取平级类 (Extract Sibling)
**原理**：如果两个类有共同点但互不包含，应提取一个更基础的基类，让两者平级。

* **案例**：`Ostrich`（鸵鸟）和 `Eagle`（鹰）都继承自 `Bird`。
* **重构**：将 `Bird` 拆分为 `FlyingBird` 和 `NonFlyingBird`。
* **结果**：`Ostrich` 继承 `NonFlyingBird`，不再被迫接受 `fly()` 的馈赠。

---

## 7. 治理决策矩阵

| 拒绝表现 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **抛出异常/空实现** | 严重违背 LSP | **必须重构**。改为组合关系或调整继承层级。 |
| **完全不使用父类方法** | “黑客式”复用代码 | **必须重构**。使用工具类 (Utils) 替代继承。 |
| **只用了 1/10 的功能** | 继承层次过宽 | **建议重构**。细化父类职责，进行接口拆分（ISP）。 |

---
