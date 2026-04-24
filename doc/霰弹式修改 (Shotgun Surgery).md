**霰弹式修改 (Shotgun Surgery)** 是一种与“发散式变化”（Divergent Change）相反但同样致命的架构坏味道。如果说发散式变化是一个类承担了太多责任，那么霰弹式修改则意味着**某种职责过于散乱**，导致原本应该内聚的一项功能被切割后撒在了系统的各个角落。

---

# 源码架构分析协议：霰弹式修改 (Shotgun Surgery)

## 1. 缺陷定义
**霰弹式修改** 指的是每当你需要对某个功能点或业务规则做出修改时，你必须在**许多不同的类或文件**中做出许多**细微的修改**。
* **物理特征**：修改范围极大（修改的文件数多），但每个文件的修改量极小（可能只是改一行代码或一个常量）。
* **架构影响**：它破坏了系统的“单点变更”能力。由于修改点散布四处，开发者很难找全所有需要改动的地方，极易遗漏，从而引入难以察觉的回归 Bug。

---

## 2. 典型场景与代码示例

### 场景：硬编码的业务策略散落在各处
假设系统需要增加一种新的“高级会员”计费逻辑。

```java
// 修改点 1: 订单类
public class Order {
    public double getTotal() {
        if (user.isVip()) return price * 0.8; // 修改
        if (user.isGoldVip()) return price * 0.7; // 新增
        return price;
    }
}

// 修改点 2: 账单报表
public class BillReport {
    public void print(User user) {
        if (user.isVip()) printVipStyle(); // 修改
        if (user.isGoldVip()) printGoldStyle(); // 新增
    }
}

// 修改点 3: 积分计算器
public class PointCalculator {
    public int calc(User user) {
        if (user.isVip()) return 10; // 修改
        if (user.isGoldVip()) return 20; // 新增
        return 5;
    }
}
```
**分析**：每增加一个会员等级，你就得像打霰弹枪一样，在全工程搜索 `isVip()` 并手动补上新的分支。这说明“会员差异化逻辑”这个职责没有被正确封装。

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别霰弹式修改通常需要结合**版本控制系统 (VCS)** 的历史数据。

| 全称 (Full Name) | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Change Coupling** | **CC** | **计算方式**：分析 Git 提交记录。如果 A 和 B 总是同时出现在同一个 Commit 中，则 CC 值高。 | $> 0.7$ |
| **Commit Scattering Degree** | **CSD** | **计算方式**：单次逻辑变更（一个 Jira/Issue）涉及的文件数量。 | $CSD > 5$ |
| **Modified Lines Ratio** | **MLR** | **计算方式**：$\frac{Modified\_Lines}{Total\_Files}$。如果比值极小，说明修改非常细碎且分散。 | $< 5$ 行/文件 |

### CSD 计算示例：
统计过去 10 次关于“计费规则”的 Commit：
* 平均涉及文件数：8 个。
* 平均每个文件修改行数：2 行。
**结论**：高 CSD + 低 MLR = 典型的霰弹式修改。

---

## 4. 缺陷命中规则 (Detection Rules)

判定核心规则：

* **规则 1：并发修改判定 (Co-change)**
    $$Rule_{CoChange} = \text{FilesInCommit} > 5 \land \text{AvgLinesPerFile} < 10$$
    *即：改动牵扯了大量文件，但每个文件都只是“蜻蜓点水”。*

* **规则 2：逻辑关联判定 (Logical Link)**
    *如果通过搜索某个关键词（如 `Enum` 值或特定 `Flag`）发现修改逻辑在不同包（Package）中重复，则命中。*

---

## 5. 检测算法伪代码实现

由于霰弹式修改在静态代码中较难察觉，通常使用“演进式分析”：

```python
def detect_shotgun_surgery(vcs_log, timespan_days=180):
    # 1. 聚类分析 Commit
    commit_clusters = group_commits_by_task_id(vcs_log)
    
    for task_id, commits in commit_clusters.items():
        affected_files = set()
        total_lines_changed = 0
        
        for c in commits:
            affected_files.update(c.files)
            total_lines_changed += c.stats.total_changes
            
        # 2. 计算分散度
        if len(affected_files) > 5:
            lines_per_file = total_lines_changed / len(affected_files)
            if lines_per_file < 5:
                report_issue("Shotgun Surgery Detected", {
                    "task": task_id,
                    "affected_files": affected_files,
                    "fragility_index": len(affected_files) / total_lines_changed
                })
```

---

## 6. 治理建议与详细案例

### 方案 A：搬移方法/字段 (Move Method/Field) —— 归拢逻辑
**原理**：将散落在各处的细碎代码提取出来，集中搬移到一个最合适的类中。
* **重构动作**：创建一个 `DiscountPolicy` 类，将所有 `if(isVip)` 的判断逻辑收编。



### 方案 B：内联类 (Inline Class)
**原理**：如果某些类太小且总是被别人连带修改，说明它没有独立存在的价值，将其并入调用者。

### 方案 C：多态取代条件表达式 (Replace Conditional with Polymorphism)
**原理**：针对上述会员等级案例，通过创建 `VipUser`、`GoldUser` 子类，将差异化行为封装在子类中。
* **效果**：新增等级只需增加一个新类，而无需修改现有的 `Order` 或 `Report` 类。

---

## 7. 治理决策矩阵

| 表现特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **多处修改相同的常量或枚举** | 缺乏集中管理 | **最高优先级**：执行“提取参数对象”或“配置中心化”。 |
| **多个类中存在类似的 Switch 判断** | 职责散乱且违反多态 | **高优先级**：执行“以多态取代条件表达式”。 |
| **修改一个类总导致另一个类报错** | 强物理耦合 | **中优先级**：执行“合并类（Inline Class）”。 |

---

**霰弹式修改是架构“刚性”的体现，治理它的过程本质上是在重新划定系统的职责边界。**