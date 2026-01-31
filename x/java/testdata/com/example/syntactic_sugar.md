### 1. 类结构与对象增强（Class Level）

这是对 QN 完整性影响最大的部分，因为它们涉及“凭空出现”的方法和字段。

* **默认构造函数 (Default Constructor)**：当类未定义构造函数时，编译器自动生成一个无参构造函数。
* **枚举增强 (Enum Implicit Methods)**：
* `values()`：返回枚举数组的静态方法。
* `valueOf(String)`：根据名称查找枚举值的静态方法。
* `static { ... }`：枚举项的初始化块。


* **Record 自动生成 (Java 14+)**：
* **Fields**：所有参数自动变为 `private final` 字段。
* **Accessors**：生成与参数名同名的 Getter（如 `id()` 而非 `getId()`）。
* **Constructor**：全参数构造函数。
* **Object Methods**：自动生成 `equals()`, `hashCode()`, `toString()`。


* **内部类引用 (Inner Class Reference)**：非静态内部类会隐式持有一个指向外部类的引用（通常名为 `this$0`）。

### 2. 函数式编程增强（Functional Style）

这部分决定了你的引用分析（References）是否能正确追踪到逻辑流。

* **方法引用 (Method References)**：
* `ClassName::staticMethod`
* `instance::instanceMethod`
* `ClassName::new` (构造函数引用)


* **Lambda 表达式 (Lambda Expressions)**：
* **单行 Lambda**：`x -> x + 1` 隐式包含了 `return` 关键字。
* **闭包捕获**：对外部 `final` 或 `effectively final` 变量的捕获。


* **接口默认/静态方法 (Default/Static Methods in Interfaces)**：接口不再只是抽象，它们可以包含具体的代码逻辑。

### 3. 控制流与语句块增强（Statement Level）

这部分会影响变量的作用域（Scope）和 QN 计数。

* **Try-with-resources**：
* 隐式调用 `AutoCloseable.close()`。
* 生成隐藏的 `catch` 块来处理关闭异常。


* **增强型 For 循环 (Enhanced For-loop)**：
* 针对集合：转换为 `Iterator` 模式。
* 针对数组：转换为下标遍历。


* **Switch 表达式 (Java 14+)**：
* `yield` 关键字返回值。
* Lambda 风格的 `->` 分支。


* **断言 (Assert Statement)**：
* `assert condition : detail;` 会被编译为 `if (!condition) throw new AssertionError(detail);`。



### 4. 类型与操作增强（Type System）

虽然它们不总是产生新的 QN，但会改变方法的调用签名。

* **自动装箱/拆箱 (Autoboxing / Unboxing)**：`Integer` 与 `int` 之间的无缝转换。
* **变长参数 (Varargs)**：将 `String...` 在内部处理为 `String[]`。
* **字符串拼接 (String Concatenation)**：`"a" + "b"` 在低版本中转为 `StringBuilder`，在高版本（Java 9+）中转为 `invokedynamic`。
* **钻石操作符 (Diamond Operator)**：`new ArrayList<>()` 中的类型推导。
* **泛型擦除 (Type Erasure)**：运行时泛型变为 `Object` 或边界类，这对我们静态分析的 QN 匹配逻辑有深远影响。

---

### 完整的语法糖概览表

| 类别 | 语法糖 / 隐式特性 | 影响的元数据 | 分析难度 |
| --- | --- | --- | --- |
| **类** | **Record 类** | 字段、方法、构造函数全量生成 | ⭐⭐⭐⭐ |
| **类** | **Enum 方法** | 增加 `values`, `valueOf` 静态方法 | ⭐⭐ |
| **类** | **默认构造函数** | 增加 `ClassName()` 定义 | ⭐ |
| **引用** | **方法引用** | 建立对已有方法/构造函数的引用关系 | ⭐⭐⭐ |
| **语句** | **Try-with-resources** | 局部变量定义、隐式方法调用 (`close`) | ⭐⭐ |
| **语句** | **foreach 循环** | 循环变量定义、隐式迭代器调用 | ⭐ |
| **函数** | **Lambda 表达式** | 匿名作用域、变量捕获关系 | ⭐⭐⭐ |
| **参数** | **变长参数** | 匹配方法签名时的类型转换 | ⭐⭐ |

### 后续计划

为了不破坏你现在的全绿测试，我建议我们分阶段实现：

1. **第一阶段：结构性补全**（补齐 Record、Enum、默认构造函数）。这保证了“**定义**”是完整的。
2. **第二阶段：引用语义增强**（补齐方法引用、Lambda 捕获、Try-with-resources 调用）。这保证了“**依赖**”是完整的。
