
---

# 源码架构分析协议：上帝类 (God Class) 

## 1. 缺陷定义
**上帝类 (God Class)** 是指那些“无所不知、无所不做的类”。它严重违反了单一职责原则（SRP），将系统的核心逻辑和大量不相关的数据强行耦合在一起。这种类通常成为系统演进的瓶颈：修改困难、测试成本高、编译时间长，且极易引入回归错误。

---

## 2. 典型场景与代码示例

### 2.1 业务逻辑黑洞 (The Logic Black Hole)
一个 `AccountService` 类，除了处理账户基本的增删改查，还负责了“信用评估”、“多币种汇率转换”、“风险控制拦截”、“短信模版渲染”等职责。
```java
public class AccountService {
    // 成员变量
    private Repo repo;
    private SmsClient sms;
    private RiskEngine risk;
    private CurrencyConvertor convertor;

    // 职责 A：核心账户操作
    public void createAccount() { ... }
    
    // 职责 B：本该属于风险模块
    public boolean checkRiskStatus(String userId) {
        // 复杂的规则引擎调用逻辑
    }

    // 职责 C：本该属于财务模块
    public double convertToUSD(double amount, String fromCurrency) {
        // 处理复杂的实时汇率计算
    }
}
```

### 2.2 万能辅助类 (The "Everything" Utils)
虽然是类，但表现为一个巨大的 `GlobalUtils` 或 `CommonHelper`，内部塞满了没有任何关联的静态方法。

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别上帝类的核心不在于代码行数，而在于以下四个维度的精密量化：

| 全称 | 简名 | 计算方式与定义 | 阈值 |
| :--- | :--- | :--- | :--- |
| **Weighted Methods per Class** | **WMC** | **计算方式**：$WMC = \sum_{i=1}^{n} c_i$。其中 $n$ 是方法数，$c_i$ 是第 $i$ 个方法的**圈复杂度**。圈复杂度基于控制流图中的节点和边，简单理解为：每出现一个 `if`、`while`、`for`、`case`，该方法的值 $+1$。 | $> 47$ |
| **Access to Foreign Data** | **ATFD** | **计算方式**：统计该类所有方法中，调用的**非自身类**的属性或 Getter 方法的去重总数。例如调用了 `User.name`、`Order.id`、`Order.price`，则 ATFD 为 2（User 和 Order）。 | $> 5$ |
| **Tight Class Cohesion** | **TCC** | **计算方式**：$TCC = \frac{NDP}{NP}$。$NP$ 为类中方法对的总数：$n(n-1)/2$；$NDP$ 为**直接连接**的方法对数（即两个方法访问了至少一个共同的成员变量）。 | $< 0.33$ |
| **Lack of Cohesion in Methods** | **LCOM** | **计算方式**：$LCOM = |P| - |Q|$（若 $|P| > |Q|$ 否则为 0）。$P$ 是不共享成员变量的方法对集合，$Q$ 是共享成员变量的方法对集合。 | 值越大越差 |

### TCC 计算详解示例：
假设类有 3 个方法 {M1, M2, M3}，成员变量有 {V1, V2}。
* M1 访问 V1
* M2 访问 V1
* M3 访问 V2
1. 所有对：(M1,M2), (M1,M3), (M2,M3)，共 **3 对**。
2. 共享对：只有 (M1,M2) 共享了 V1，共 **1 对**。
3. **TCC** = 1 / 3 = **0.33**。

---

## 4. 缺陷命中规则 (Detection Rules)

判定上帝类的黄金法则（Lanza & Marinescu 规则）：

### 规则 1：上帝类触发器 (God Formula)
当一个类在逻辑复杂度、外部数据依赖、内部缺乏内聚三个维度同时超标时：
$$Rule_{God} = (WMC > 47) \land (ATFD > 5) \land (TCC < 0.33)$$

### 规则 2：职责集中度警告 (Concentration)
如果一个类的方法数量超过包（Package）中所有方法总数的 1/3。
$$Rule_{Volume} = \frac{Methods(Class)}{Methods(Package)} > 0.33$$

---

## 5. 检测算法伪代码实现

```python
def detect_god_class(source_tree):
    for cls in source_tree.classes:
        # 1. 计算 WMC (加权方法复杂度)
        wmc = 0
        for m in cls.methods:
            wmc += calculate_cyclomatic_complexity(m)
            
        # 2. 计算 ATFD (访问外部数据数)
        external_entities = set()
        for m in cls.methods:
            external_entities.update(find_external_references(m))
        atfd = len(external_entities)
        
        # 3. 计算 TCC (紧密内聚度)
        all_pairs = list(combinations(cls.methods, 2))
        connected_pairs = 0
        for m1, m2 in all_pairs:
            if set(get_used_fields(m1)) & set(get_used_fields(m2)):
                connected_pairs += 1
        tcc = connected_pairs / len(all_pairs) if all_pairs else 1
        
        # 4. 规则匹配
        if wmc > 47 and atfd > 5 and tcc < 0.33:
            report_issue(cls, "God Class", {"WMC": wmc, "ATFD": atfd, "TCC": tcc})
```

---

## 6. 治理建议与详细案例

### 方案 A：提取类 (Extract Class) —— 解决低内聚
**原理**：根据 TCC 计算中发现的“孤岛方法”，将不共享变量的方法簇移动到新类。
* **案例**：`AccountService` 中的汇率转换逻辑。
* **重构前**：`AccountService` 既查余额又算汇率。
* **重构后**：提取 `CurrencyService`，`AccountService` 通过依赖注入调用它。
* **结果**：`AccountService` 的 $TCC$ 从 0.25 提升到 0.6。



### 方案 B：移动方法 (Move Method) —— 解决高 ATFD
**原理**：如果一个方法访问外部数据比访问内部数据还多，就该把方法迁走。
* **案例**：`OrderManager` 内部有一个 `calculateDiscount(User u)` 方法，全是 `u.getGrade()`、`u.getPoints()`。
* **重构**：将该逻辑移至 `User` 类或专门的 `DiscountCalculator`。
* **结果**：$ATFD$ 显著下降，符合“迪米特法则”。

### 方案 C：门面模式 (Facade Pattern) —— 治理中心化
**原理**：如果无法立即拆分，先通过 Facade 将上帝类拆分为多个子模块，上帝类仅作为流量入口。

---

## 7. 治理决策矩阵

| 指标表现 | 根本原因 | 建议重构动作 |
| :--- | :--- | :--- |
| **WMC 极高，但 TCC 也高** | 职责单一但过于笨重 | **提取子类/状态模式**。将复杂的 `if-else` 分支转化为多态实现。 |
| **TCC < 0.2** | 类内部存在多个无关逻辑堆积 | **最高优先级：提取类**。强制按变量共享关系拆分为 2-3 个小类。 |
| **ATFD > 10** | 典型的“依恋情节” | **中优先级：移动方法**。将逻辑归还给数据的所有者。 |

---