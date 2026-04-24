**继承层次混乱 (Hierarchy Complexity)** 是衡量面向对象系统健康度的重要宏观指标。一个健康的继承体系应该是“倒金字塔”或“平衡树”状的。当继承层次变得过深（纵向过度耦合）或单层过宽（横向职责不清）时，系统的可理解性和维护性会呈指数级下降。

---

# 源码架构分析协议：继承层次混乱 (Hierarchy Complexity)

## 1. 缺陷定义
**继承层次混乱** 主要表现为两种极端形态：
1.  **过深继承 (Deep Hierarchy)**：继承链条过长，导致开发者难以追踪属性和方法的来源，修改顶层逻辑会引发不可预知的链式反应。
2.  **贫乏继承 (Anemic/Thin Hierarchy)**：每一层只有一个子类（即单线继承），这种结构往往是为了“预留扩展性”而进行的过度设计，增加了不必要的间接性。

---

## 2. 典型场景与代码示例

### 2.1 递归式深层继承 (The Deep Crawler)
逻辑被分散在 5 层以上的继承链中，查找一个方法的最终实现需要跨越多个文件。
```java
class BaseView { ... }
class AbstractButton extends BaseView { ... }
class ClickableButton extends AbstractButton { ... }
class StyledButton extends ClickableButton { ... }
class RoundStyledButton extends StyledButton { ... }
// 缺陷：如果要在 RoundStyledButton 中调试点击逻辑，
// 必须在 5 个类之间反复横跳，理解负担极大。
```

### 2.2 “保姆式”单线继承 (Speculative Hierarchy)
为了所谓的“未来扩展性”，为每个具体的类都强行套上一个父类，导致类数量翻倍但逻辑并未解耦。
```java
interface IService { void run(); }
abstract class AbstractService implements IService { ... }
public class RealService extends AbstractService { 
    // 缺陷：整个系统中只有 RealService 一个实现。
    // AbstractService 成了多余的“中间人”。
}
```

---

## 3. 抽象度量指标 (Metrics)

为了量化继承树的健康度，我们引入以下标准 OO 指标：

| 全称 (Full Name) | 简名 | 描述 | 计算公式 |
| :--- | :--- | :--- | :--- |
| **Depth of Inheritance Tree** | **DIT** | **继承深度**。该类到根节点的路径长度。 | 建议上限：5 |
| **Number of Children** | **NOC** | **子类数量**。该类直接拥有的子类个数。 | 衡量父类的抽象负担 |
| **Specialization Index** | **SIX** | **特化索引**。衡量继承带来的新行为多还是重写多。 | $SIX = \frac{Level \times N_{new}}{N_{total}}$ |
| **Weighted Methods per Class** | **WMC** | **类加权方法数**。反映类本身的复杂度。 | 用于判断职责是否下沉过快 |

---

## 4. 缺陷命中规则 (Detection Rules)

### 规则 1：深层嵌套判定 (Deep Jungle)
**定义**: 继承层级超过了人类短期记忆的极限（通常认为 5 层是红线）。
$$Rule_{Deep} = DIT > 5$$

### 规则 2：单线冗余判定 (Skeleton Hierarchy)
**定义**: 存在多层继承，但每一层的分支因子（NOC）都等于 1。
$$Rule_{Skeleton} = (DIT \ge 3) \land (\forall \text{ Ancestor } A: NOC(A) == 1)$$

### 规则 3：宽而浅的结构 (Wide & Shallow)
**定义**: 单个父类的子类过多，导致父类成为变更热点。
$$Rule_{Wide} = NOC > 15$$

---

## 5. 检测算法伪代码实现

### 第一阶段：构建全局继承树
```python
def build_hierarchy_tree(all_classes):
    tree = Tree()
    for cls in all_classes:
        tree.add_node(cls.id, parent=cls.super_class)
    return tree
```

### 第二阶段：深度与宽度扫描
```python
def analyze_hierarchy_complexity(tree):
    issues = []
    for node in tree.all_nodes():
        dit = tree.get_depth(node)
        noc = tree.get_children_count(node)
        
        if dit > 5:
            issues.append({"target": node, "type": "TOO_DEEP", "val": dit})
        
        # 检测单线继承
        if dit >= 3:
            path = tree.get_path_to_root(node)
            if all(tree.get_children_count(p) == 1 for p in path[1:-1]):
                issues.append({"target": node, "type": "SINGLE_LINE_REDUNDANCY"})
                
    return issues
```

---

## 6. 治理建议与详细案例

### 方案 A：以组合替代深层继承 (Flattening with Composition)
**原理**：将深层继承中的中间层逻辑拆分为独立的“装饰器”或“策略对象”。

* **案例**：`RoundStyledButton` 继承链。
* **重构**：创建一个 `Button` 类，持有 `Shape`（圆/方）和 `Style`（颜/色）属性。
* **结果**：DIT 从 5 降为 1。通过配置不同的属性组合出各种按钮。


### 方案 B：塌陷继承体系 (Collapse Hierarchy)
**原理**：如果父类与其唯一的子类之间没有实质性的职责差异，将它们合并为一个类。

* **案例**：`AbstractService` 与 `RealService`。
* **重构**：将 `AbstractService` 的逻辑合并进 `RealService` 并删除父类。
* **结果**：减少了代码跳转层级，提高了可读性。

---

## 7. 治理决策矩阵

| 表现特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **DIT > 5 且方法分散** | 纵向过度耦合 | **最高优先级**：执行“组合替代继承” (Replace Inheritance with Delegation)。 |
| **DIT > 3 且 NOC 全为 1** | 过度设计 | **中优先级**：执行“塌陷继承体系” (Collapse Hierarchy)。 |
| **NOC 过大（根节点过重）** | 职责划分不当 | **中优先级**：执行“提取中间类” (Extract Superclass) 或将子类归类。 |

---