这是一个为您整理的完整、专业级《循环依赖（Circular Dependency）抽象与检测方案》。

---

# 源码架构分析协议：循环依赖 (Circular Dependency)

## 1. 缺陷定义
**循环依赖 (Circular Dependency)** 指两个或多个模块（类、文件、包）之间形成的闭环依赖关系，违反了 **非循环依赖原则 (ADP)**。它会导致系统耦合度激增、难以进行单元测试、无法独立重用模块，甚至在某些框架中导致初始化死锁。

---

## 2. 典型场景与代码示例

### 2.1 物理层面的直接循环 (Hard Cycle)
最显性的循环，通常表现为成员属性的互相持有。
```java
// 类 A 引用 B
public class ServiceA {
    private ServiceB serviceB; // 成员变量依赖
}

// 类 B 引用 A
public class ServiceB {
    private ServiceA serviceA; 
}
```

### 2.2 逻辑层面的隐性调用 (Implicit Logic Cycle)
不通过成员变量，而是通过方法调用、参数传递或局部变量实例化形成的环。
```java
// 场景：ReportGenerator -> Utils -> ReportGenerator
public class ReportGenerator {
    public void generate() {
        Utils.format(this); // 通过静态方法或参数传递 A -> B
    }
}

public class Utils {
    public static void format(ReportGenerator rg) {
        new ExportTask().run(rg); // B 内部产生了对 A 的依赖
    }
}
```

### 2.3 继承体系的下溯依赖 (Inheritance Downward Cycle)
父类（抽象）依赖了其具体的子类实现。
```java
// 基类 Shape.java
public abstract class Shape {
    public static Shape createCircle() {
        return new Circle(); // 违背原则：Shape -> Circle
    }
}

// 子类 Circle.java
public class Circle extends Shape { } // Circle -> Shape (继承依赖)
```

### 2.4 跨包/模块级循环 (Package-Level Cycle)
类之间没有直接循环，但所属的包之间存在环路，导致包无法独立发布。
* **Package P1**: 包含类 `A1`，`A1` 引用了 `Package P2` 中的 `B1`。
* **Package P2**: 包含类 `B2`，`B2` 引用了 `Package P1` 中的 `A2`。
* **结果**: `P1 <--> P2` 形成环。

---

## 3. 抽象度量指标 (Metrics)

| 全称 (Full Name) | 简名 | 描述 | 示例/取值 |
| :--- | :--- | :--- | :--- |
| **Strongly Connected Component** | **SCC** | 布尔值，标识图中是否存在强连通分量（即闭环）。 | `true` / `false` |
| **Node Count in SCC** | **N_SCC** | 处于同一个闭环中的节点（类或文件）数量。 | $N\_SCC \ge 2$ |
| **Dependency Edge Type** | **DET** | 依赖边的性质。 | `Inherit`, `Field`, `Param`, `Invoke` |
| **Edge Weight** | **EW** | 依赖强度权重。 | 继承(10), 成员变量(8), 局部调用(1) |
| **Dependency Path Length** | **DPL** | 环路中闭环路径经过的节点总数（跨度）。 | $A \to B \to C \to A, DPL=3$ |
| **Afferent Coupling** | **Ca** | 扇入：有多少外部节点依赖此环路节点。 | 衡量环路节点的“稳定性” |
| **Cycle Weight Sum** | **CWS** | 环路中所有边的 EW 总和。 | 用于重构优先级的量化评估 |

---

## 4. 缺陷命中规则 (Detection Rules)

通过指标构建逻辑公式，自动识别不同严重程度的循环依赖：

### 规则 1：物理耦合环 (Critical)
**定义**: 环路由继承或成员变量组成，通常导致无法通过静态编译或引起内存/启动死锁。
$$Rule_{Hard} = (SCC == true) \land (N\_SCC > 1) \land \forall e \in Cycle: (DET(e) \in \{Inherit, Field\})$$

### 规则 2：逻辑穿透环 (Warning)
**定义**: 环路中包含方法调用或局部变量，虽能编译，但破坏了架构的层次感。
$$Rule_{Soft} = (SCC == true) \land \exists e \in Cycle: (DET(e) \in \{Invoke, Param, Local\_Var\})$$

### 规则 3：架构枢纽环 (Fatal)
**定义**: 处于高扇入（Ca）核心地位的模块参与了循环。
$$Rule_{Hub} = Rule_{Hard} \land (Ca(Node) > \text{Threshold}_{stable})$$

---

## 5. 检测算法伪代码实现

### 第一阶段：多维依赖图建模

```python
def build_graph(ast_nodes):
    G = DirectedGraph()
    for node in ast_nodes:
        # 1. 识别继承
        for parent in node.super_types:
            G.add_edge(node.id, parent, type='Inherit', weight=10)
        # 2. 识别成员属性
        for field in node.fields:
            G.add_edge(node.id, field.type, type='Field', weight=8)
        # 3. 识别方法体内的隐性调用
        for method in node.methods:
            for call in method.calls:
                # 需结合符号表 resolve 类型
                target_type = symbol_table.resolve(call.target)
                G.add_edge(node.id, target_type, type='Invoke', weight=1)
    return G
```

### 第二阶段：环路识别 (基于 Tarjan 算法)
```python
def detect_scc(graph):
    # 使用 Tarjan 算法查找强连通分量
    sccs = tarjan_algorithm(graph)
    detected_issues = []
    
    for component in sccs:
        if len(component) > 1:
            # 计算指标
            metrics = {
                "N_SCC": len(component),
                "CWS": sum(e.weight for e in get_edges_in(component)),
                "DPL": calculate_max_path(component)
            }
            # 规则匹配
            if metrics["CWS"] > THRESHOLD_FATAL:
                status = "CRITICAL"
            else:
                status = "WARNING"
            
            detected_issues.append({"nodes": component, "status": status, "metrics": metrics})
    return detected_issues
```

---

## 6. 治理建议 与 详细案例

### 方案 A：依赖倒置 (Dependency Inversion, DIP) —— 针对“实现类”循环
**适用场景**：类 A 需要调用类 B 的具体功能，而类 B 又需要反向引用 A。
**原理**：引入抽象接口，使依赖箭头从“横向”或“反向”变为“向上”指向抽象。

* **详细案例**：
    * **现状**：`OrderService`（订单）调用 `PaymentService`（支付）进行扣款；扣款成功后，`PaymentService` 需要回调 `OrderService.updateStatus()` 更新订单状态。形成 `OrderService <-> PaymentService`。
    * **治理**：定义一个接口 `PaymentCallback`，让 `OrderService` 实现它。
    * **结果**：`PaymentService` 只依赖 `PaymentCallback` 接口，而不再依赖具体的 `OrderService`。


---

### 方案 B：提取公用逻辑 (Extract Common Component) —— 针对“工具类”循环
**适用场景**：两个类 A 和 B 互相引用，是因为它们都包含了一部分对方需要的公共逻辑或数据结构。
**原理**：将重叠部分提取到第三个独立的类（或底层包） C 中。

* **详细案例**：
    * **现状**：类 `User` 包含 `getAddressString()`（依赖 `Address` 类）；而 `Address` 类包含 `formatForUser(User u)`（依赖 `User` 类）。
    * **治理**：创建一个 `AddressFormatter` 类，将格式化逻辑移入其中。
    * **结果**：`User` 和 `Address` 都依赖 `AddressFormatter`（或由 `AddressFormatter` 依赖它们），消除两者间的直接闭环。

---

### 方案 C：中介者模式 (Mediator Pattern) —— 针对“复杂网状”循环
**适用场景**：多个类（A, B, C, D...）之间存在极其复杂的互相调用。
**原理**：禁止类之间直接通信，所有交互通过一个“中介者”转发。

* **详细案例**：
    * **现状**：在一个复杂的 GUI 表单中，`TextBox` 变动会影响 `Button` 状态，`Button` 点击会触发 `Checkbox` 检查，`Checkbox` 又会清空 `TextBox`。
    * **治理**：引入 `FormMediator` 控制器。
    * **结果**：所有组件只向 `FormMediator` 发送事件，由中介者决定下一步操作。组件间依赖度降为 0。


---

### 方案 D：事件驱动/发布订阅 (Event-Driven) —— 针对“业务流程”循环
**适用场景**：上层业务逻辑与下层通知逻辑形成的循环。
**原理**：使用消息队列或事件总线（EventBus），将“硬调用”改为“发送信号”。

* **详细案例**：
    * **现状**：`InventoryService`（库存）在缺货时调用 `PurchaseService`（采购）；`PurchaseService` 在采购入库后调用 `InventoryService.increase()`。
    * **治理**：`PurchaseService` 完成后发布 `ItemArrivedEvent`。
    * **结果**：`PurchaseService` 不再知道 `InventoryService` 的存在，循环被切断。

---

### 方案 E：动态注入/延迟加载 (Lazy Loading) —— 针对“运行时”循环
**适用场景**：无法规避的构造函数循环依赖（常见于遗留系统或 Spring 框架中）。
**原理**：通过延迟初始化（如 `ObjectProvider` 或 `@Lazy`），使对象先完成实例化，在首次使用时再注入依赖。

* **详细案例**：
    * **现状**：类 A 构造函数需要 B，类 B 构造函数需要 A。系统启动报 `BeanCurrentlyInCreationException`。
    * **治理**：在 B 的构造函数参数前加 `@Lazy`。
    * **结果**：Spring 会先给 A 注入一个 B 的代理对象，等真正调用 B 的方法时才去查找 B，避开了启动时的死锁。

---

## 7. 治理决策矩阵

为了方便选择方案，可参考下表：

| 循环类型 | 首选方案 | 重构成本 | 收益 |
| :--- | :--- | :--- | :--- |
| **基础实现循环** | 方案 A (DIP/接口化) | 低 | 极高，符合面向对象原则 |
| **逻辑重叠循环** | 方案 B (提取公用类) | 中 | 提高代码复用率，职责内聚 |
| **复杂网状耦合** | 方案 C (中介者) | 高 | 极大降低复杂系统的维护成本 |
| **跨模块/异步业务** | 方案 D (事件驱动) | 高 | 实现架构解耦，增强系统伸缩性 |
| **遗留代码/框架限制** | 方案 E (延迟加载) | 极低 | 快速止血，但不解决本质耦合 |

---
