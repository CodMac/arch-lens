**数据类 (Data Class)** 是“功能依恋”的孪生坏味道。如果说功能依恋是“逻辑找错了家”，那么数据类就是“家徒四壁”——它只提供存放数据的容器，没有任何业务行为。这种类通常沦为其他类（如上帝类或功能依恋类）的“数据桶”，导致系统的逻辑极度分散。

---

# 源码架构分析协议：数据类 (Data Class)

## 1. 缺陷定义
**数据类** 指的是那些几乎只包含字段（Fields）以及用于访问这些字段的 Getter 和 Setter 的类。它们没有复杂的行为，不参与业务逻辑的计算。
* **危害**：这种类会导致“贫血模型（Anemic Domain Model）”，业务逻辑会溢出到其他的 Service 或 Manager 中，使得原本内聚的领域知识变得碎片化。
* **例外**：DTO（数据传输对象）或纯粹的配置映射类在特定架构下是允许的。

---

## 2. 典型场景与代码示例

### 2.1 哑对象 (The Dumb Object)
类中没有任何逻辑方法，仅作为数据的载体。
```java
public class UserAccount {
    private String username;
    private double balance;
    private int status;

    // 只有标准的 Getter 和 Setter
    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
    public double getBalance() { return balance; }
    public void setBalance(double balance) { this.balance = balance; }
    // ... 其他 Setter/Getter ...
}
```

### 2.2 暴露细节的容器 (Exposed Internals)
类中虽然有一些方法，但这些方法只是简单地返回内部集合或状态，没有封装性可言。

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

数据类的检测核心在于评估**“数据密度”**与**“行为密度”**的失衡。

| 全称 | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Number of Public Attributes** | **NOPA** | **计算方式**：统计类中公开字段的数量 + 公开的 Getter/Setter 数量。 | $> 5$ |
| **Weight of Class** | **WOC** | **计算方式**：$WOC = \frac{N_{functional}}{N_{total}}$。即“非访问方法”占“总公开方法”的比例。 | $< 0.33$ |
| **Weighted Methods per Class** | **WMC** | **计算方式**：类中所有方法的圈复杂度总和（见前文）。数据类的 WMC 通常极低。 | $< 10$ |

### WOC 计算示例：
类中有 10 个公开方法，其中 8 个是 Getter/Setter，2 个是业务逻辑方法。
* $N_{functional} = 2$
* $N_{total} = 10$
* **$WOC = 2 / 10 = 0.2$** (说明该类 80% 的职责是暴露数据)

---

## 4. 缺陷命中规则 (Detection Rules)

判定数据类需要排除 DTO 等特殊用途类，主要针对核心业务包中的类。

### 规则 1：贫血模型判定 (Anemic Rule)
方法极多但几乎全是访问器，逻辑复杂度极低。
$$Rule_{Anemic} = (NOPA > 5) \land (WOC < 0.33) \land (WMC < 10)$$

### 规则 2：被动数据持有者 (Passive Holder)
如果该类同时被多个具有“功能依恋”倾向的类频繁访问。

---

## 5. 检测算法伪代码实现

```python
def detect_data_class(class_nodes):
    for cls in class_nodes:
        # 1. 统计公开属性和访问器 (NOPA)
        public_attrs = cls.get_public_fields()
        accessors = [m for m in cls.methods if m.is_getter() or m.is_setter()]
        nopa = len(public_attrs) + len(accessors)
        
        # 2. 计算行为权重 (WOC)
        functional_methods = [m for m in cls.methods if not (m.is_getter() or m.is_setter())]
        woc = len(functional_methods) / len(cls.methods) if cls.methods else 0
        
        # 3. 计算复杂度 (WMC)
        wmc = sum(calculate_cc(m) for m in cls.methods)
        
        # 4. 规则匹配
        if nopa > 5 and woc < 0.33 and wmc < 10:
            # 进一步检查是否有功能依恋类在“压榨”它
            envy_clients = find_feature_envy_clients(cls)
            report_issue(cls, "Data Class", {"WOC": woc, "EnvyClients": len(envy_clients)})
```

---

## 6. 治理建议与详细案例

### 方案 A：封装并移动行为 (Encapsulate and Move) —— 核心治理
**原理**：寻找那些“依恋”该数据类的方法，将它们移入数据类中。
* **案例**：`OrderManager` 频繁读取 `UserAccount` 的余额并判断是否足够。
* **重构**：在 `UserAccount` 中增加 `canAfford(amount)` 方法。
* **结果**：数据类有了“灵魂”，$WOC$ 指标提升。



### 方案 B：封装字段 (Encapsulate Field)
**原理**：如果字段是公开的，先将其设为私有，并检查所有的 Setter 是否可以被更有意义的业务方法替代。
* **案例**：`user.setStatus(2)`。
* **重构**：改为 `user.deactivate()` 或 `user.freeze()`。

---

## 7. 治理决策矩阵

| 指标表现 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **WOC 极低且在核心包中** | 逻辑外泄，贫血模型 | **最高优先级**：执行“移动方法”，让逻辑回归数据。 |
| **WOC 极低且在协议/传输层** | 正常的 DTO/VO | **无需重构**：它们本身就是为了传输数据设计的。 |
| **NOPA 巨大且无业务方法** | 数据库表的直接映射 | **建议重构**：检查该类是否承载了过重的业务含义，尝试将其拆分为更小的领域模型。 |

---