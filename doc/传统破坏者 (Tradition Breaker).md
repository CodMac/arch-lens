**传统破坏者 (Tradition Breaker)** 是对里氏替换原则（LSP）的直接挑战。如果说“拒绝父类馈赠”是子类在行为上表现得像个“叛逆者”，那么“传统破坏者”就是子类在**契约**和**可见性**上表现得像个“破坏者”。它通常不仅拒绝功能，还试图改变基类已经定义好的“传统”（即公共协议）。

---

# 源码架构分析协议：传统破坏者 (Tradition Breaker)

## 1. 缺陷定义
**传统破坏者** 指子类在继承过程中，通过技术手段削弱了父类的契约能力。这种破坏主要体现在两个维度：
1.  **可见性退化**：将父类的 `public` 或 `protected` 方法改为更私有的权限，导致原本针对父类的调用在子类实例上失效。
2.  **契约空心化**：虽然保留了方法名，但内部实现被清空或改为无意义的返回，且未通过异常告知调用者，这种“静默破坏”比显式的抛出异常更难调试。

---

## 2. 典型场景与代码示例

### 2.1 削弱可见性 (Visibility Weakening)
这种行为直接违反了 OO 语言的继承规范（在某些语言如 Java 中会编译报错，但在 C++ 或通过某些技巧绕过检查的情况下会发生）。
```java
// 父类定义了公共接口
public class BaseService {
    public void executeTask() { /* 逻辑 */ }
}

// 子类试图“隐藏”这个能力
public class PrivateSubService extends BaseService {
    // 缺陷：将 Public 变为 Private (或通过不重写但隐藏的方式)
    // 使得：BaseService s = new PrivateSubService(); s.executeTask(); 报错
    private void executeTask() { /* ... */ } 
}
```

### 2.2 契约背叛：虚假的重写 (Empty Override)
子类继承了父类的关键逻辑方法，却通过重写将其功能置为空，导致原本依赖该逻辑的父类流程在子类中失效。
```java
public class DataSyncer {
    public void validate() { /* 执行复杂的安全校验 */ }
    public final void sync() { 
        validate(); // 父类流程依赖 validate
        save(); 
    }
}

public class LazySyncer extends DataSyncer {
    @Override
    public void validate() {
        // 传统破坏：直接置空。父类以为执行了校验，实际跳过了。
    }
}
```

---

## 3. 抽象度量指标 (Metrics)

| 全称 (Full Name) | 简名 | 描述 | 计算公式 |
| :--- | :--- | :--- | :--- |
| **Visibility Degradation Count** | **VDC** | **可见性退化数**。子类中可见性低于父类的同名方法数。 | 计数 |
| **Nop Override Ratio** | **NOR** | **空操作重写率**。重写方法中仅含 `return` 或无代码的比例。 | $NOR = \frac{N_{empty}}{N_{overridden}}$ |
| **Contract Breach Degree** | **CBD** | **契约破坏度**。子类修改了父类预设状态（Pre-condition）的程度。 | 定性分析/符号执行 |

---

## 4. 缺陷命中规则 (Detection Rules)

### 规则 1：可见性退化判定 (LSP Violation)
**定义**: 子类方法可见性 < 父类方法可见性。
$$Rule_{Visibility} = \exists m \in SubClass: Visibility(m) < Visibility(SuperClass.m)$$

### 规则 2：空洞重写判定 (Hollow Contract)
**定义**: 重写了父类具有逻辑意义的方法，但内容为空。
$$Rule_{Hollow} = (NOR > 0) \land (\text{MethodType} \in \{Logic, Safety, Setup\})$$
*(注：需排除掉父类本身就是 abstract 且子类是首次实现的情况)*

---

## 5. 检测算法伪代码实现

### 第一阶段：权限对比扫描
```python
def detect_visibility_breach(sub_class, super_class):
    breaches = []
    for m_sub in sub_class.methods:
        m_super = super_class.find_method(m_sub.name)
        if m_super and is_less_accessible(m_sub.visibility, m_super.visibility):
            breaches.append(m_sub.name)
    return breaches
```

### 第二阶段：方法体内聚性分析
```python
def detect_hollow_overrides(class_node):
    # 查找所有带有 @Override 且非抽象的方法
    overrides = [m for m in class_node.methods if m.is_override and not m.is_abstract]
    hollow_methods = []
    
    for m in overrides:
        # 检查是否只有空行、注释或无意义的 return
        if is_effectively_empty(m.body):
            hollow_methods.append(m.name)
            
    return len(hollow_methods) / len(overrides) if overrides else 0
```

---

## 6. 治理建议与详细案例

### 方案 A：使用 Template Method 模式并加锁
**原理**：父类将核心流程设为 `final`，只暴露必要的钩子（Hook）给子类重写，防止子类破坏主流程。

* **案例**：上面的 `DataSyncer`。
* **重构**：将 `sync()` 设为 `final`。将 `validate()` 设为 `protected` 或提供默认实现。
* **结果**：子类可以重写校验逻辑，但无法破坏“必须经过校验”这个传统。



### 方案 B：由继承转为委派 (Delegation)
**原理**：如果子类由于某种原因必须让某些方法失效，说明它不具备父类的类型特征，应停止继承。

* **案例**：`PrivateSubService` 不想暴露父类接口。
* **重构**：删除继承关系，在 `PrivateSubService` 内部持有一个 `BaseService` 实例。
* **结果**：只暴露自己想暴露的方法，完全控制了接口的访问权限。

---

## 7. 治理决策矩阵

| 破坏特征 | 诊断结论 | 治理动作 |
| :--- | :--- | :--- |
| **可见性退化** | 语法层/设计层严重错误 | **必须立即重构**。取消继承，改为组合。 |
| **核心逻辑空实现** | 隐性 Bug 诱因 | **重构父类**。使用 `final` 固定流程，或将方法设为 `abstract` 强制子类思考实现。 |
| **仅为禁用功能** | 错误的抽象归类 | **重新分类**。寻找更小的共同基类，或引入接口拆分（ISP）。 |

---