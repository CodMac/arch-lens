**功能依恋 (Feature Envy)** 是一种典型的类间耦合缺陷。它描述了一种“放错位置”的逻辑：一个类的方法似乎对另一个类的数据更感兴趣，而不是它所在的类。这种依恋破坏了**封装性**，导致数据和处理逻辑被物理隔离。

---

# 源码架构分析协议：功能依恋 (Feature Envy)

## 1. 缺陷定义
**功能依恋** 指的是某个方法频繁地访问另一个类的数据（通常是通过调用大量的 Getter 方法），而几乎不使用其所属类的方法或属性。
* **现象**：方法 A 属于类 Class1，但它 80% 的操作都在和 Class2 交互。
* **本质**：逻辑与数据分离，违反了“高内聚、低耦合”的原则。

---

## 2. 典型场景与代码示例

### 2.1 “过度关心”他人的数据 (Data Greed)
方法内部充斥着对另一个对象的连续调用，本类却沦为了一个“搬运工”。
```java
public class OrderService {
    // 缺陷：calculateDiscount 对 User 的内部细节过于依恋
    public double calculateDiscount(User user) {
        // 全是在调用 user 的数据
        if (user.getAge() > 60 && user.getMembershipLevel() > 3) {
            return user.getPoints() * 0.1;
        }
        return 0;
    }
}
```

### 2.2 远程操纵逻辑 (Remote Control)
一个类的方法在控制另一个类的状态转换，而另一个类仅作为简单的“数据桶（Data Bucket）”。

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别功能依恋需要对比方法对**“自身”**与**“外部”**的访问频率。

| 全称 | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Access to Foreign Data** | **ATFD** | **计算方式**：该方法访问的外部类属性/Getter 的数量（去重）。 | $> 3$ |
| **Locality of Access** | **LA** | **计算方式**：$LA = \frac{Access(Self)}{Access(Self) + Access(Foreign)}$。衡量访问的本地化比例。 | $< 0.33$ |
| **Foreign Data Providers** | **FDP** | **计算方式**：该方法所依恋的外部类的数量。如果只依恋一个类，重构最容易。 | $1$ |

### LA 计算示例：
方法 `M` 访问了自己类的 1 个变量，访问了 `User` 类的 4 个属性。
* $Access(Self) = 1$
* $Access(Foreign) = 4$
* **$LA = 1 / (1 + 4) = 0.2$** (说明 80% 的兴趣在外部)

---

## 4. 缺陷命中规则 (Detection Rules)

判定功能依恋需要满足“外向性”和“冷淡性”两个条件：

### 规则 1：依恋判定 (Envy Rule)
方法访问外部数据多，且对自身类兴趣极低。
$$Rule_{Envy} = (ATFD > 3) \land (LA < 0.33) \land (FDP \le 2)$$

### 规则 2：依恋目标单一性
如果一个方法同时依恋 5、6 个类，那它可能不是功能依恋，而是“上帝方法”，重点在于拆分而非移动。
$$Rule_{Target} = FDP \le 2$$

---

## 5. 检测算法伪代码实现

```python
def detect_feature_envy(methods):
    for m in methods:
        # 1. 统计对本类成员的访问
        self_access = count_references(m, m.parent_class)
        
        # 2. 统计对外部类的访问
        foreign_access_map = {} # {Class_B: count, Class_C: count}
        for ref in m.get_external_references():
            target_class = ref.target_class
            foreign_access_map[target_class] = foreign_access_map.get(target_class, 0) + 1
            
        atfd = sum(foreign_access_map.values())
        fdp = len(foreign_access_map)
        
        # 3. 计算本地化比例 LA
        la = self_access / (self_access + atfd) if (self_access + atfd) > 0 else 1
        
        # 4. 规则匹配
        if atfd > 3 and la < 0.33 and fdp <= 2:
            envy_target = max(foreign_access_map, key=foreign_access_map.get)
            report_issue(m, "Feature Envy", {"Target": envy_target, "LA": la})
```

---

## 6. 治理建议与详细案例

### 方案 A：移动方法 (Move Method) —— 最优解
**原理**：将依恋他人的方法物理迁移到被依恋的类中。
* **案例**：`OrderService.calculateDiscount(User u)`。
* **重构**：将该方法移入 `User` 类。
* **结果**：`calculateDiscount` 变成了 `User` 的内部逻辑。`LA` 瞬间从 0.2 变为 1.0。



### 方案 B：提取方法并移动 (Extract & Move)
**原理**：如果方法中只有一部分依恋他人，先提取这部分，再将其移走。
* **案例**：方法前 10 行在算账，后 20 行在疯狂读取 `User` 信息。
* **重构**：将后 20 行提取为 `User.getDiscountInfo()`，原方法改为调用新方法。

---

## 7. 治理决策矩阵

| 指标表现 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **LA < 0.1 且 FDP = 1** | 严重的单点依恋 | **最高优先级**：执行 **Move Method**。这个逻辑根本不该在这。 |
| **ATFD 很高但 FDP 很大** | “交际花”方法 | **中优先级**：执行 **Extract Class**。它可能正在承担某种协调者的职责。 |
| **LA 较高 ( > 0.5)** | 正常的协作 | **无需重构**。类之间总会有交互。 |

---
