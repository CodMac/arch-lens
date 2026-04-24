**紧耦合 (Intensive Coupling)** 是在方法层面对系统内聚性的深度挑战。它与“依恋情节”不同：依恋情节关注的是对**数据**（Getter/Fields）的渴求，而紧耦合关注的是对**行为**（外部方法）的深度缠绕。当一个方法通过密集的调用逻辑与多个外部对象交织在一起时，它就变成了一个脆弱的连接点。

---

# 源码架构分析协议：紧耦合 (Intensive Coupling)

## 1. 缺陷定义
**紧耦合** 指的是某个方法不仅依赖于外部类，而且通过频繁且密集的调用与这些外部类的方法深度绑定。
* **核心特征**：方法内部逻辑是通过协调多个外部对象的方法来实现的，且这种调用往往是跨领域的。
* **本质矛盾**：该方法缺乏足够的抽象，直接操作了过多的外部实现细节。这导致该方法成为了系统的“脆弱支点”——任何一个被调用类的接口变化或逻辑重构，都会直接导致该方法的崩溃。

---

## 2. 典型场景与代码示例

### 场景：缺乏中介者的“协调者”方法
在一个电商结算系统中，一个结算方法直接控制了库存、积分、支付和物流等多个模块的具体行为。

```java
// 示例：PaymentProcessor.java
// 现象：processPayment 方法与多个外部类的方法密集交织
public class PaymentProcessor {
    public void processPayment(Order order) {
        // 紧耦合：直接操作了 4 个不相关的外部领域方法
        if (inventoryService.checkStock(order.getId())) { 
            double price = pricingEngine.calculateFinalPrice(order);
            paymentGateway.authorize(order.getUser(), price);
            
            // 密集的外部调用
            pointSystem.addPoints(order.getUser(), price);
            logisticsProvider.createShipment(order);
            emailService.sendConfirmation(order.getUser());
        }
    }
}
```

---

## 3. 抽象度量指标：深度量化与计算方式 (Metrics)

识别紧耦合需要分析方法调用的**广度**与**强度**。

| 全称 (Full Name) | 简名 | 计算方式与定义 | 阈值参考 |
| :--- | :--- | :--- | :--- |
| **Coupling Intensity** | **CINT** | **计算方式**：该方法调用的**不同外部方法**的总数（去重后）。 | $CINT > 7$ |
| **Coupling Dispersion** | **CDISP** | **计算方式**：$CDISP = \frac{Dist(Providers)}{CINT}$。其中 $Dist$ 是提供这些方法的外部类数量。反映耦合的“散乱”程度。 | $CDISP < 0.25$ (说明集中耦合于少数类) |
| **Method Call Density** | **MCD** | **计算方式**：$\frac{External\_Calls}{LOC}$。每行代码包含的外部调用密度。 | 偏离均值 |

### CINT 与 CDISP 计算示例：
在上述 `processPayment` 方法中：
1. **外部调用 (CINT)**: `checkStock`, `calculateFinalPrice`, `authorize`, `addPoints`, `createShipment`, `sendConfirmation` $\rightarrow$ **CINT = 6**。
2. **提供者 (Dist)**: `InventoryService`, `PricingEngine`, `PaymentGateway`, `PointSystem`, `LogisticsProvider`, `EmailService` $\rightarrow$ **Dist = 6**。
3. **CDISP**: $6 / 6 = 1.0$ (说明耦合非常分散，典型的“霰弹式依恋”)。

---

## 4. 缺陷命中规则 (Detection Rules)

判定紧耦合的复合规则：

* **规则 1：高强度耦合判定**
    $$Rule_{Intensity} = (CINT > 7) \land (CDISP < 0.3)$$
    *即：方法密集地调用了少数几个外部类的方法（深层缠绕）。*

* **规则 2：分散式紧耦合判定**
    $$Rule_{Dispersion} = (CINT > 5) \land (CDISP > 0.7)$$
    *即：方法像“章鱼”一样，触角伸向了全系统各个不同的领域类（职责模糊）。*

---

## 5. 检测算法伪代码实现

```python
def detect_intensive_coupling(method_node):
    external_calls = method_node.get_external_method_calls() # 获取所有外部调用
    
    # 1. 统计调用强度 CINT
    cint = len(set(external_calls)) 
    
    # 2. 统计外部提供者类
    providers = set([call.target_class for call in external_calls])
    dist_providers = len(providers)
    
    # 3. 计算分散度 CDISP
    cdisp = dist_providers / cint if cint > 0 else 0
    
    # 4. 逻辑判定
    if cint > 7:
        if cdisp < 0.3:
            report_issue("Intensive Coupling (Deep)", {"CINT": cint, "CDISP": cdisp})
        elif cdisp > 0.7:
            report_issue("Intensive Coupling (Wide/Shotgun)", {"CINT": cint, "CDISP": cdisp})
```

---

## 6. 治理建议与详细案例

### 方案 A：引入中介者 (Introduce Mediator) / 门面模式 (Facade)
**原理**：如果一个方法协调了太多外部对象，说明缺少一个高层级的抽象来管理这种协作。
* **重构动作**：创建一个 `OrderOrchestrator`，将库存、支付、物流的协作逻辑封装进去。
* **效果**：`PaymentProcessor` 只需调用 `Orchestrator` 的一个方法，`CINT` 显著下降。



### 方案 B：依赖倒置 (Dependency Inversion)
**原理**：不要直接依赖具体的实现类，而是定义接口。
* **案例**：不再直接调用 `EmailService` 和 `SmsService`，而是发布一个 `OrderPaidEvent`，让订阅者自行处理后续。

### 方案 C：提取并搬移 (Extract & Move)
**原理**：如果一段连续的外部调用都在操作同一个外部对象，将这段逻辑提取并搬移到该对象中。

---

## 7. 治理决策矩阵

| 表现特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **CINT 高且 CDISP 很低** | 与少数几个类“过度亲密” | **最高优先级**：检查是否应该将逻辑合并或执行“移动方法”。 |
| **CINT 高且 CDISP 很高** | 职责过于发散，充当了乱序协调者 | **高优先级**：执行“引入门面（Facade）”或“事件驱动解耦”。 |
| **CINT 在合理范围但代码行数极少** | 纯委托方法 | **无需重构**：除非导致了过长的消息链。 |

---

**紧耦合的治理目标是让每个方法“各司其职”，只与必要的抽象层进行对话。这能极大地提升系统的可测试性，因为你不再需要为了测试一个方法而 Mock 全世界的对象。**