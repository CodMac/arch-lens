package model

// --- 依赖关系类型 (Dependency Relation Types) ---

// DependencyType 是表示依赖关系的字符串常量
type DependencyType string

const (
	// --- 1. 组织与逻辑层级关系 (Structural & Hierarchy) ---

	// Import 导入: 源码文件引用了外部包/类
	// e.g., [Java: Source(File) -> Target(Package/Class/Constant)]
	Import DependencyType = "IMPORT"

	// Export 导出: 模块或文件对外暴露的符号 (常见于 JS/TS, Go)
	// e.g., [JS: Source(File) -> Target(Function/Variable)]
	Export DependencyType = "EXPORT"

	// Contain 包含: 物理或逻辑上的归属关系，用于构建全局符号树
	// e.g., [Java: Source(Class) -> Target(Method/Field)]
	Contain DependencyType = "CONTAIN"

	// --- 2. 面向对象继承与实现 (Inheritance & Implementation) ---

	// Extend 继承: 类与类、接口与接口之间的继承
	// e.g., [Java: Source(Class/Interface) -> Target(Class/Interface)]
	Extend DependencyType = "EXTEND"

	// Implement 实现: 类实现接口
	// e.g., [Java: Source(Class) -> Target(Interface)]
	Implement DependencyType = "IMPLEMENT"

	// Override 重写: 子类方法对父类/接口方法的具体实现或覆盖
	// e.g., [Java: Source(Method) -> Target(Method)]
	Override DependencyType = "OVERRIDE"

	// --- 3. 类型绑定与元数据 (Typing & Metadata) ---

	// Annotation 注解: 符号被特定注解修饰
	// e.g., [Java: Source(Class/Method) -> Target(AnnotationClass)]
	Annotation DependencyType = "ANNOTATION"

	// Parameter 参数类型: 方法/函数参数对类型的依赖
	// e.g., [Java: Source(Method) -> Target(Type/Class)]
	Parameter DependencyType = "PARAMETER"

	// Return 返回类型: 方法返回值对类型的依赖
	// e.g., [Java: Source(Method) -> Target(Type/Class)]
	Return DependencyType = "RETURN"

	// Throw 抛出异常: 方法声明抛出的异常类型
	// e.g., [Java: Source(Method) -> Target(Class)]
	Throw DependencyType = "THROW"

	// TypeArg 泛型实参: 变量定义中泛型参数指向的类型
	// e.g., [Java: Source(List<User>) -> Target(User)]
	TypeArg DependencyType = "TYPE_ARG"

	// --- 4. 行为与执行流 (Behavioral & Execution) ---

	// Call 调用: 函数或方法的显式/隐式调用
	// e.g., [Java: Source(Method) -> Target(Method)]
	Call DependencyType = "CALL"

	// Create 显式实例创建: 源码中通过 new 关键字实例化对象
	// 关注点：谁触发了实例化动作，或哪个变量持有了新创建的实例
	// e.g., [Java: Source(Method/Field/Variable) -> Target(Class)]
	Create DependencyType = "CREATE"

	// Cast 强转: 显式的类型转换
	// e.g., [Java: Source(Expression) -> Target(Class)]
	Cast DependencyType = "CAST"

	// --- 5. 数据流与状态引用 (Data Flow & State) ---

	// Use 使用: 读取变量、字段或常量的值
	// e.g., [Java: Source(Method) -> Target(Field/Variable)]
	Use DependencyType = "USE"

	// Assign 赋值: 将值写入变量或字段
	// e.g., [Java: Source(Expression) -> Target(Variable/Field)]
	Assign DependencyType = "ASSIGN"

	// Capture 捕获: 闭包（Lambda/匿名内部类）对外部局部变量的引用
	// e.g., [Java: Source(Lambda) -> Target(LocalVariable)]
	Capture DependencyType = "CAPTURE"

	// --- 6. 跨语言或特殊链接 (Special Linking) ---

	// ImplLink 符号链接: 用于 C/C++ 等区分声明与实现的语言，将原型链接到定义
	ImplLink DependencyType = "IMPL_LINK"

	// Mixin 混入: 动态将一个模块的功能注入另一个类
	Mixin DependencyType = "MIXIN"
)

// DependencyRelation 是工具的核心输出结构，描述了 Source 和 Target 之间的一个依赖关系
type DependencyRelation struct {
	// Type: 依赖关系的类型 (e.g., CALL, IMPORT, EXTEND)
	Type DependencyType `json:"Type"`

	// Source: 关系的发起方（调用者、导入者、子类等）
	Source *CodeElement `json:"Source"`

	// Target: 关系的指向方（被调用者、被导入者、父类等）
	Target *CodeElement `json:"Target"`

	// Location: 关系发生的代码位置。
	// 对于 Call 关系，这是调用点；对于 Import，则是 import 语句所在行。
	Location *Location `json:"Location"`

	// Details: 存储关系的附加描述（如具体的调用签名、是否为静态调用等）
	Details string `json:"Details,omitempty"`

	// Mores: 预留的扩展属性字典
	Mores map[string]interface{} `json:"Mores,omitempty"`
}
