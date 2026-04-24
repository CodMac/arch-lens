**精神分裂类 (Schizophrenic Class)** 是对单一职责原则（SRP）最严重的背离之一。如果说“上帝类”是因为管得太宽而变得臃肿，那么“精神分裂类”则是由于在同一个类中强行缝合了两个或多个**互不相关**的逻辑人格，导致类在不同的使用场景下表现出完全不同的行为特征。

---

# 源码架构分析协议：精神分裂类 (Schizophrenic Class)

## 1. 缺陷定义
**精神分裂类** 是指一个类提供了一组在逻辑上、业务上或上下文上完全不相关的接口或功能。
* **核心特征**：类内部存在明显的“逻辑断层”。
* **内在矛盾**：类中的方法分为几簇，每一簇只访问特定的成员变量，簇与簇之间几乎没有任何交集。这使得该类看起来像是一个被迫挤在同一个物理空间里的多个小对象的集合。

---

## 2. 典型场景与代码示例

### 场景：强行归类的“混合管理中心”
开发者为了省事，将“用户权限校验”和“系统日志归档”这两个完全不同的职责塞进了同一个类中。

```java
// 示例：SystemIntegrator.java
// 现象：类内部存在两个完全不重叠的职责簇
public class SystemIntegrator {
    // 成员变量簇 A：权限相关
    private Map<String, List<String>> permissionCache;
    private AuthService authClient;

    // 成员变量簇 B：日志/硬件监控相关
    private File logDirectory;
    private DiskScanner scanner;

    // --- 人格 A：身份验证逻辑 ---
    public boolean validateUser(String token) {
        // 仅使用变量簇 A
        return authClient.verify(token, permissionCache);
    }

    public void updateCache() {
        // 仅使用变量簇 A
        permissionCache.putAll(authClient.fetchLatest());
    }

    // --- 人格 B：系统维护逻辑 ---
    public void archiveOldLogs(int days) {
        // 仅使用变量簇 B
        scanner.findFilesOlderThan(days, logDirectory).forEach(File::delete);
    }

    public long getAvailableSpace() {
        // 仅使用变量簇 B
        return logDirectory.getFreeSpace();
    }
}
```

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别“精神分裂”的关键在于量化**方法与属性之间的连接密度**。

| 全称 (Full Name) | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Disjoint Sets count** | **DSC** | **计算方式**：将方法视为节点，若两个方法共享至少一个成员变量，则连一条边。计算最终生成的**连通分量**数量。 | $DSC > 1$ |
| **Lack of Cohesion in Methods** | **LCOM** | **计算方式**：$LCOM = \|P\| - \|Q\|$。$P$ 为不共享变量的方法对数量，$Q$ 为共享变量的方法对数量。 | $> 10$ (取决于方法总数) |
| **Overlap Ratio** | **OR** | **计算方式**：$OR = \frac{\text{共享变量的方法对}}{\text{总方法对}}$。 | $< 0.1$ |

### DSC（不相交集合数）计算过程：
1.  列出所有方法：$\{M1, M2, M3, M4\}$。
2.  观察访问关系：
    * $M1, M2 \rightarrow \{V1\}$（形成集合 1）
    * $M3, M4 \rightarrow \{V2, V3\}$（形成集合 2）
3.  **结论**：由于集合 1 和集合 2 没有任何连边，**DSC = 2**。这意味着类已经分裂为两个独立的人格。

---

## 4. 缺陷命中规则 (Detection Rules)

判定精神分裂类的组合规则：

* **规则 1：物理分裂判定**
    $$Rule_{Split} = DSC \ge 2$$
    *只要 DSC 大于 1，说明类在逻辑上已经是断裂的。*

* **规则 2：低内聚高耦合判定**
    $$Rule_{Cohesion} = (LCOM > 20) \land (\text{Methods} > 6)$$
    *当方法较多且它们之间几乎不共享任何数据时，通常暗示了隐性的精神分裂。*

---

## 5. 检测算法伪代码实现

```python
def detect_schizophrenic_class(cls):
    methods = cls.get_all_methods()
    fields = cls.get_all_fields()
    
    # 1. 构建邻接矩阵或图
    # 节点是方法，边表示它们访问了共同的字段
    adj_matrix = build_shared_field_graph(methods, fields)
    
    # 2. 使用并查集或 BFS 查找连通分量
    connected_components = find_clusters(adj_matrix)
    dsc = len(connected_components)
    
    # 3. 输出结果
    if dsc > 1:
        return True, {
            "is_schizophrenic": True,
            "number_of_personalities": dsc,
            "clusters": connected_components # 返回每个“人格”包含的方法列表
        }
    return False, None
```

---

## 6. 治理建议与详细案例

### 方案 A：提取类 (Extract Class) —— 核心方案
**原理**：根据检测算法识别出的连通分量（Clusters），将每个簇物理拆分为一个独立的类。
* **案例**：将 `SystemIntegrator` 拆分为 `UserAuthService` 和 `SystemMaintenanceService`。
* **重构动作**：
    1.  创建新类。
    2.  搬移属于该人格的成员变量。
    3.  使用 **Move Method** 将相关方法迁入。
    4.  原类如果需要保留，可通过组合（Composition）方式持有新类的实例。



### 方案 B：接口隔离 (Interface Segregation)
**原理**：如果由于历史原因暂时无法拆分类，应至少通过不同的接口来隔离职责。
* **案例**：定义 `IAuthenticator` 和 `IMaintainer` 两个接口，让 `SystemIntegrator` 同时实现它们，但在调用侧只暴露所需的接口。

---

## 7. 治理决策矩阵

| 表现特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **DSC >= 2 且变量不重叠** | 明确的职责分裂 | **最高优先级**：执行“提取类”。这是最容易且收益最高的重构。 |
| **DSC = 1 但 LCOM 很高** | 弱内聚（可能存在上帝类倾向） | **中优先级**：检查是否某些方法访问了过多的变量，考虑“提取方法”。 |
| **多个人格且互相调用** | 混乱的紧耦合分裂 | **高优先级**：先解耦调用逻辑，再进行类提取。 |

---
