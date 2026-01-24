package model

// --- 代码元素类型 (Code Element Kinds) ---

// ElementKind 是表示代码实体类型的字符串常量
type ElementKind string

const (
	File            ElementKind = "FILE"             // 基本结构体          	-> 源文件
	Package         ElementKind = "PACKAGE"          // 基本结构体        	-> 包 (Go, Java, C++)
	Module          ElementKind = "MODULE"           // 结构化代码块      	-> 模块 (Python, Rust)
	Namespace       ElementKind = "NAMESPACE"        // 结构化代码块      	-> 命名空间 (C++, C#)
	AnonymousClass  ElementKind = "ANONYMOUS_CLASS"  // 面向对象/复合类型      -> 匿名内部类 (Java)
	Class           ElementKind = "CLASS"            // 面向对象/复合类型    	-> 类 (Java, C++, Python, JS)
	Interface       ElementKind = "INTERFACE"        // 面向对象/复合类型    	-> 接口 (Java, Go, TS)
	AnonymousStruct ElementKind = "ANONYMOUS_STRUCT" // 面向对象/复合类型      -> 匿名结构体 (Go)
	Struct          ElementKind = "STRUCT"           // 面向对象/复合类型    	-> 结构体 (Go, C)
	Enum            ElementKind = "ENUM"             // 面向对象/复合类型    	-> 枚举 (Java, C++, Rust)
	EnumConstant    ElementKind = "ENUM_CONSTANT"    // 面向对象/复合类型    	-> 枚举常量
	KAnnotation     ElementKind = "ANNOTATION"       // 面向对象/复合类型    	-> 注解 (Java, C++, Python, JS)
	Trait           ElementKind = "TRAIT"            // 面向对象/复合类型    	-> 特质 (Rust, Scala)
	Lambda          ElementKind = "LAMBDA"           // 可执行体    			-> Lambda 表达式 (Java)
	MethodRef       ElementKind = "METHOD_REF"       // 可执行体    			-> 方法引用 (Java)
	ScopeBlock      ElementKind = "SCOPE_BLOCK"      // 可执行体    			-> 代码块 (用于分析局部变量作用域)
	Function        ElementKind = "FUNCTION"         // 可执行体  	     	-> 独立函数 (C, Go, JS)
	Method          ElementKind = "METHOD"           // 可执行体       		-> 类/结构体的方法
	Macro           ElementKind = "MACRO"            // 可执行体       		-> 预处理器宏 (C/C++)
	Variable        ElementKind = "VARIABLE"         // 存储和声明         	-> 局部/全局变量
	Constant        ElementKind = "CONSTANT"         // 存储和声明         	-> 常量
	Field           ElementKind = "FIELD"            // 存储和声明         	-> 类/结构体/枚举的成员或字段 (Java, Go, C++)
	Type            ElementKind = "TYPE"             // 存储和声明          	-> 自定义类型别名或基本类型引用
	Unknown         ElementKind = "UNKNOWN"          // 未知类型
)

// Location 描述了代码元素或依赖关系在源码中的位置
type Location struct {
	FilePath    string `json:"FilePath"`
	StartLine   int    `json:"StartLine"`
	EndLine     int    `json:"EndLine"`
	StartColumn int    `json:"StartColumn"`
	EndColumn   int    `json:"EndColumn"`
}

// CodeElement 描述了源码中的一个可识别实体（Source 或 Target）
type CodeElement struct {
	Kind           ElementKind `json:"Kind"`                // Kind: 元素的类型 (e.g., FUNCTION, CLASS, VARIABLE)
	Name           string      `json:"Name"`                // Name: 元素的短名称 (e.g., "main", "CalculateSum")
	QualifiedName  string      `json:"QualifiedName"`       // QualifiedName: 元素的完整限定名称 (e.g., "pkg/util.Utility.CalculateSum")
	Path           string      `json:"Path"`                // Path: 元素所在的文件路径 (相对于项目根目录)
	Signature      string      `json:"Signature,omitempty"` // Signature: 元素的完整签名（针对函数/方法，包含参数和返回值类型）
	Location       *Location   `json:"Location,omitempty"`  // Location: 元素的位置
	Doc            string      `json:"Doc,omitempty"`       // Doc: 文档注释 (如 Javadoc, Go Doc)
	Comment        string      `json:"Comment,omitempty"`   // Comment: 普通注释 (行/块注释)
	Extra          *Extra      `json:"Extra,omitempty"`     // Extra 额外信息
	IsFormSource   bool        `json:"IsFormSource"`        // 是否源码产生的实体
	IsFormSugar    bool        `json:"IsFormSugar"`         // 是否语法糖产生的实体
	IsFormExternal bool        `json:"IsFormExternal"`      // 是否外部导入产生的实体
}

// Extra CodeElement的额外信息。包含了跨语言（如Java和Go）通用的关键元数据。
type Extra struct {
	Modifiers   []string               `json:"Modifiers,omitempty"`  // 修饰符列表 (e.g., "public", "private", "static", "final", "abstract")
	Annotations []string               `json:"Annotation,omitempty"` // 注解列表 (e.g., "@Service")
	Mores       map[string]interface{} `json:"Mores,omitempty"`      // Mores 存储特定语言或特定类型的额外属性 (如 Java 的 Throws, Go 的 Receiver)
}
