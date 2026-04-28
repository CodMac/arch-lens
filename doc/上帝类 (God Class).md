
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
| **Weighted Methods per Class** | **WMC** | **计算方式**：$WMC = \sum_{i=1}^{n} c_i$。每出现一个 `if`、`while`、`for`、`case`，值 $+1$。 | $> 47$ |
| **Access to Foreign Data** | **ATFD** | **计算方式**：统计该类所有方法中，调用的**非自身类**的属性或 Getter 方法的去重总数。需排除内部类。 | $> 5$ |
| **Tight Class Cohesion** | **TCC** | **计算方式**：$TCC = \frac{NDP}{NP}$。$NP$ 为方法对总数 $n(n-1)/2$；$NDP$ 为访问了共同成员变量的方法对数。 | $< 0.33$ |
| **Lack of Cohesion in Methods** | **LCOM** | **计算方式**：$LCOM = \|P\| - \|Q\|$。$P$ 为不共享变量的方法对，$Q$ 为共享变量的方法对。 | 值越大越差 |
| **Weighted Methods per Class** | **WMC** | **计算方式**：$WMC = \sum_{i=1}^{n} c_i$。每出现一个 `if`、`while`、`for`、`case`，值 $+1$。 | $> 47$ |
| **Access to Foreign Data** | **ATFD** | **计算方式**：统计该类所有方法中，调用的**非自身类**的属性或 Getter 方法的去重总数。需排除内部类。 | $> 5$ |
| **Tight Class Cohesion** | **TCC** | **计算方式**：$TCC = \frac{NDP}{NP}$。$NP$ 为方法对总数 $n(n-1)/2$；$NDP$ 为访问了共同成员变量的方法对数。 | $< 0.33$ |
| **Lack of Cohesion in Methods** | **LCOM** | **计算方式**：$LCOM = \|P\| - \|Q\|$。$P$ 为不共享变量的方法对，$Q$ 为共享变量的方法对。 | 值越大越差 |

### 3.1 WMC 计算实例
假设 `OrderProcessor` 有两个方法：
* **方法 A**: 包含 1 个 `if` 和 1 个 `for`。复杂度 $c_1 = 1（基础）+ 1 + 1 = 3$。
* **方法 B**: 包含一个带有 5 个 `case` 的 `switch`。复杂度 $c_2 = 1 + 5 = 6$。
* **WMC 结果**：$3 + 6 = 9$。

### 3.2 ATFD 计算实例（含排除逻辑）
```java
public class PriceCalculator {
    // 内部类视为职责延伸，访问它不计入 ATFD
    private class InternalHelper { public double getBase() { return 10.0; } }

    public void calc(User u, Product p) {
        double rate = u.getVIPRate();      // 访问外部类 User (ATFD +1)
        double base = p.getBasePrice();    // 访问外部类 Product (ATFD +1)
        double offset = new InternalHelper().getBase(); // 访问内部类 (排除)
    }
}
```
* **ATFD 结果**：涉及外部实体个数为 **2**。

### 3.3 TCC 计算实例
### 3.1 WMC 计算实例
假设 `OrderProcessor` 有两个方法：
* **方法 A**: 包含 1 个 `if` 和 1 个 `for`。复杂度 $c_1 = 1（基础）+ 1 + 1 = 3$。
* **方法 B**: 包含一个带有 5 个 `case` 的 `switch`。复杂度 $c_2 = 1 + 5 = 6$。
* **WMC 结果**：$3 + 6 = 9$。

### 3.2 ATFD 计算实例（含排除逻辑）
```java
public class PriceCalculator {
    // 内部类视为职责延伸，访问它不计入 ATFD
    private class InternalHelper { public double getBase() { return 10.0; } }

    public void calc(User u, Product p) {
        double rate = u.getVIPRate();      // 访问外部类 User (ATFD +1)
        double base = p.getBasePrice();    // 访问外部类 Product (ATFD +1)
        double offset = new InternalHelper().getBase(); // 访问内部类 (排除)
    }
}
```
* **ATFD 结果**：涉及外部实体个数为 **2**。

### 3.3 TCC 计算实例
假设类有 3 个方法 {M1, M2, M3}，成员变量有 {V1, V2}。
* M1 访问 V1；M2 访问 V1；M3 访问 V2。
1. **所有对 (NP)**：(M1,M2), (M1,M3), (M2,M3)，共 **3 对**。
2. **共享对 (NDP)**：仅 (M1,M2) 共享 V1，共 **1 对**。
3. **TCC** = $1 / 3 \approx 0.33$。
* M1 访问 V1；M2 访问 V1；M3 访问 V2。
1. **所有对 (NP)**：(M1,M2), (M1,M3), (M2,M3)，共 **3 对**。
2. **共享对 (NDP)**：仅 (M1,M2) 共享 V1，共 **1 对**。
3. **TCC** = $1 / 3 \approx 0.33$。

---

## 4. 特殊场景约束说明 (Constraints)
## 4. 特殊场景约束说明 (Constraints)

* **约束 A（DTO/POJO）**：若 90% 以上方法为简单的 Getter/Setter，归类为“数据类”，不作为上帝类处理。
* **约束 B（代码生成）**：强制跳过包含 `@Generated` 注解或位于 `target/generated-sources` 路径下的类。
* **约束 C（历史门面类）**：若方法逻辑仅为转发（Delegation），该方法的 WMC 权重降为 0。
* **约束 D（测试类）**：测试类 ATFD 阈值放宽 3 倍。
* **约束 E（内部类隔离）**：**ATFD 计算必须排除宿主类与内部类之间的互访**。内部类被视为宿主类的职责延伸，除非该内部类被大量第三方外部类直接依赖（入度异常）。

---

## 5. 缺陷命中规则 (Detection Rules)
* **约束 A（DTO/POJO）**：若 90% 以上方法为简单的 Getter/Setter，归类为“数据类”，不作为上帝类处理。
* **约束 B（代码生成）**：强制跳过包含 `@Generated` 注解或位于 `target/generated-sources` 路径下的类。
* **约束 C（历史门面类）**：若方法逻辑仅为转发（Delegation），该方法的 WMC 权重降为 0。
* **约束 D（测试类）**：测试类 ATFD 阈值放宽 3 倍。
* **约束 E（内部类隔离）**：**ATFD 计算必须排除宿主类与内部类之间的互访**。内部类被视为宿主类的职责延伸，除非该内部类被大量第三方外部类直接依赖（入度异常）。

---

## 5. 缺陷命中规则 (Detection Rules)

### 规则 1：上帝类触发器 (God Formula)
$$Rule_{God} = (WMC > 47) \land (ATFD > 5) \land (TCC < 0.33)$$

### 规则 2：职责集中度警告 (Concentration)
$$Rule_{Volume} = \frac{Methods(Class)}{Methods(Package)} > 0.33$$

---

## 6. 治理建议与详细案例

### 方案 A：提取类 (Extract Class) —— 解决低内聚
**原理**：根据 TCC 计算中发现的“孤岛方法”，将不共享变量的方法簇移动到新类。
* **案例**：`AccountService` 中的汇率转换逻辑。
* **重构前**：`AccountService` 既查余额又算汇率。
* **重构后**：提取 `CurrencyService`，`AccountService` 通过依赖注入调用它。

### 方案 B：移动方法 (Move Method) —— 解决高 ATFD
**原理**：如果一个方法访问外部数据比访问内部数据还多，就该把方法迁走。
* **案例**：`OrderManager.calculateDiscount(User u)` 频繁调用 `u` 的属性。
* **重构**：将该逻辑移至 `User` 类或专门的评分计算器。

### 方案 C：策略模式 / 状态模式 (Strategy/State Pattern) —— 解决高复杂度
**原理**：针对 $WMC$ 极高但 $TCC$ 正常的场景。此时类职责尚算单一，但逻辑分支（`if-else` / `switch`）爆炸。
* **案例**：一个 `OrderStateProcessor` 处理 10 种订单状态的流转逻辑。
* **重构**：定义 `OrderState` 接口，将每种状态的逻辑封装在独立的 `State` 实现类中。
* **结果**：$WMC$ 被平摊到多个子类，主类仅负责状态切换。
* **案例**：`OrderManager.calculateDiscount(User u)` 频繁调用 `u` 的属性。
* **重构**：将该逻辑移至 `User` 类或专门的评分计算器。

### 方案 C：策略模式 / 状态模式 (Strategy/State Pattern) —— 解决高复杂度
**原理**：针对 $WMC$ 极高但 $TCC$ 正常的场景。此时类职责尚算单一，但逻辑分支（`if-else` / `switch`）爆炸。
* **案例**：一个 `OrderStateProcessor` 处理 10 种订单状态的流转逻辑。
* **重构**：定义 `OrderState` 接口，将每种状态的逻辑封装在独立的 `State` 实现类中。
* **结果**：$WMC$ 被平摊到多个子类，主类仅负责状态切换。

### 方案 D：门面模式 (Facade Pattern) —— 治理中心化
### 方案 D：门面模式 (Facade Pattern) —— 治理中心化
**原理**：如果无法立即拆分，先通过 Facade 将上帝类拆分为多个子模块，上帝类仅作为流量入口。

---

## 7. 治理决策矩阵

| 指标表现 | 根本原因 | 建议重构动作 |
| :--- | :--- | :--- |
| **WMC 极高，但 TCC 也高** | 职责单一但逻辑极其繁琐 | **策略模式/状态模式**：拆分复杂的嵌套分支。 |
| **TCC < 0.2** | 内部存在多个逻辑孤岛 | **提取类 (Extract Class)**：按成员变量共享关系强行拆分。 |
| **ATFD > 10** | 典型的“依恋情节” | **移动方法 (Move Method)**：将逻辑归还给数据的所有者。 |
| **三项指标全面漂红** | 系统性设计失败 | **门面模式 (Facade)**：先建立代理入口，再逐步剥离。 |

---

## 8. 检测算法伪代码实现

```python
def detect_god_class(source_tree):
    for cls in source_tree.classes:
        if is_dto_or_generated(cls): continue # 触发约束过滤
        
        # 1. 计算 WMC
        wmc = sum(calculate_cyclomatic_complexity(m) for m in cls.methods)
        
        # 2. 计算 ATFD (包含内部类排除逻辑)
        external_entities = set()
        for m in cls.methods:
            for ref in find_external_references(m):
                # 排除自身、内部类、以及宿主类
                if not is_internal_relation(cls, ref):
                    external_entities.add(ref)
        atfd = len(external_entities)
        
        # 3. 计算 TCC
        all_pairs = list(combinations(cls.methods, 2))
        ndp = sum(1 for m1, m2 in all_pairs if set(used_fields(m1)) & set(used_fields(m2)))
        tcc = ndp / len(all_pairs) if all_pairs else 1
        
        # 4. 判定逻辑
        if wmc > 47 and atfd > 5 and tcc < 0.33:
            report_issue(cls, "God Class", {"WMC": wmc, "ATFD": atfd, "TCC": tcc})

def is_internal_relation(current_cls, target_cls):
    # 逻辑：判定 target 是否是 current 的内部类，或 current 是否是 target 的内部类
    return target_cls == current_cls or \
           target_cls in current_cls.inner_classes or \
           current_cls.parent_class == target_cls
```

---

**ArchLens 协议规范 - #07 上帝类治理方案**
