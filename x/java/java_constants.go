package java

// =============================================================================
// java_collector 采集的 Element.Extra.Mores
// =============================================================================

const (
	BlockIsStatic                  = "java.block.is_static"           // 代码块是否为静态初始化块 (static { ... }) -> bool
	ClassIsStatic                  = "java.class.is_static"           // 类是否为静态内部类 -> bool
	ClassIsAbstract                = "java.class.is_abstract"         // 类是否为抽象类 (abstract) -> bool
	ClassIsFinal                   = "java.class.is_final"            // 类是否为终态类 (final) -> bool
	ClassSuperClass                = "java.class.superclass"          // 类的父类名称 (含泛型) -> string
	ClassImplementedInterfaces     = "java.class.interfaces"          // 类实现的接口列表 -> []string
	InterfaceImplementedInterfaces = "java.interface.interfaces"      // 接口继承的其他接口列表 -> []string
	MethodIsConstructor            = "java.method.is_constructor"     // 是否为构造方法 -> bool
	MethodIsDefault                = "java.method.is_default"         // 是否为接口中的默认方法 (default) -> bool
	MethodIsImplicit               = "java.method.is_implicit"        // 是否为隐式方法 (如编译器自动生成的构造函数) -> bool
	MethodIsAnnotation             = "java.method.is_annotation"      // 是否为注解类型中的元素方法 -> bool
	MethodDefaultValue             = "java.method.default_value"      // 注解元素的默认值 -> string
	MethodReturnType               = "java.method.return_type"        // 方法返回值的原始类型文本 -> string
	MethodParameters               = "java.method.parameters"         // 方法定义的参数列表 (含类型和名称) -> []string
	MethodThrowsTypes              = "java.method.throws"             // 方法声明抛出的异常类型列表 -> []string
	VariableType                   = "java.variable.type"             // 局部变量的类型文本 -> string
	VariableIsFinal                = "java.variable.is_final"         // 局部变量是否带有 final 修饰符 -> bool
	VariableIsParam                = "java.variable.is_param"         // 变量是否为方法参数 -> bool
	FieldType                      = "java.field.type"                // 成员字段的类型文本 -> string
	FieldIsStatic                  = "java.field.is_static"           // 字段是否为静态 (static) -> bool
	FieldIsFinal                   = "java.field.is_final"            // 字段是否为终态 (final) -> bool
	FieldIsConstant                = "java.field.is_constant"         // 字段是否为常量 (通常指 static final 且有初始值) -> bool
	FieldIsRecordComponent         = "java.field.is_record_component" // 字段是否为 Java Record 的组件成员 -> bool
	MethodRefReceiver              = "java.method_ref.receiver"       // 方法引用中的接收者 (如 System.out::println 中的 System.out) -> string
	MethodRefTarget                = "java.method_ref.target"         // 方法引用中的目标方法名 (如 println) -> string
	MethodRefTypeArgs              = "java.method_ref.type_arguments" // 方法引用中显式指定的泛型参数 -> string
	EnumArguments                  = "java.enum.arguments"            // 枚举常量定义时传入的构造参数列表 -> []string
	LambdaParameters               = "java.lambda.parameters"         // Lambda 表达式的参数定义文本 -> string
	LambdaBodyIsBlock              = "java.lambda.is_block"           // Lambda 主体是否为大括号包裹的代码块 -> bool
)

// =============================================================================
// java_extractor 采集的 Relation.Mores
// =============================================================================

// --- Core属性 (java_extractor 默认提取的属性) ---
const (
	RelRawText               = "java.rel.raw_text"                // 完整的赋值语句源码 (eg,: data.name = "Hi")
	RelAstKind               = "java.rel.ast_kind"                // 触发该关系的那个 AST 节点的类型 (eg,: assignment_expression)
	RelContext               = "java.rel.context"                 // 该动作发生的大环境或语句容器 (eg,: expression_statement 或 method_declaration)
	RelCallReceiver          = "java.rel.call.receiver"           // 谁发起的调用 (如 "this", "super", 或变量名)
	RelCallReceiverType      = "java.rel.call.receiver_type"      // 发起调用的静态类名 (QN)
	RelCallIsStatic          = "java.rel.call.is_static"          // 是否为静态方法调用
	RelCallIsConstructor     = "java.rel.call.is_constructor"     // 是否为构造函数调用 (new 或 explicit this/super)
	RelCallIsChained         = "java.rel.call.is_chained"         // 是否属于调用链的一部分
	RelCallIsFunctional      = "java.rel.call.is_functional"      // 是否为方法引用 (eg,: this::simpleMethod)
	RelCallEnclosingMethod   = "java.rel.call.enclosing_method"   // 调用发生的外部方法 QN (常用于 Lambda/内部类溯源)
	RelAssignTargetName      = "java.rel.assign.target_name"      // 赋值语句目标 (谁被改变了)
	RelAssignOperator        = "java.rel.assign.operator"         // 赋值运算符，如 "=", "+=", "++"
	RelAssignIsInitializer   = "java.rel.assign.is_initializer"   // 是否为声明时的初始化赋值 (如 int i = 0)
	RelAssignValueExpression = "java.rel.assign.value_expression" // 赋值语句右侧的原始表达式文本
	RelAssignIsCapture       = "java.rel.assign.is_capture"       // 是否为跨作用域的变量捕获赋值
	RelAssignEnclosingMethod = "java.rel.assign.enclosing_method" // 赋值发生的外部方法 QN (常用于 Lambda/内部类溯源)
	RelCreateIsArray         = "java.rel.create.is_array"         // 是否为数组实例化
	RelCreateVariableName    = "java.rel.create.variable_name"    // 接收实例化对象的变量名
	RelUseReceiver           = "java.rel.use.receiver"            // 实例字段访问的接收者 (如 "this")
	RelUseIsCapture          = "java.rel.use.is_capture"          // 是否为跨作用域的变量捕获引用
	RelUseEnclosingMethod    = "java.rel.use.enclosing_method"    // 引用发生的外部方法 QN (常用于 Lambda/内部类溯源)
	RelAnnotationTarget      = "java.rel.annotation.target"       //
	RelThrowIndex            = "java.rel.throw.index"             // 在 throws 声明列表中的位置索引 (从 0 开始)
	RelThrowIsSignature      = "java.rel.throw.is_signature"      // 是否为方法签名中 throws 关键字后的声明
	RelThrowIsRuntime        = "java.rel.throw.is_runtime"        // 是否为主动抛出的运行时异常 (RuntimeException 及其子类)
	RelThrowIsRethrow        = "java.rel.throw.is_rethrow"        // 是否为重新抛出已存在的异常对象 (如 throw e)
	RelParameterIndex        = "java.rel.parameter.index"         // 参数在方法签名中的从 0 开始的索引位置
	RelParameterName         = "java.rel.parameter.name"          // 参数定义的名称 (如 "id", "name")
	RelParameterIsVarargs    = "java.rel.parameter.is_varargs"    // 是否为可变参数 (Object... args)
	RelReturnIsPrimitive     = "java.rel.return.is_primitive"     // 返回类型是否为 Java 基础类型 (int, byte, etc.)
	RelReturnIsArray         = "java.rel.return.is_array"         // 返回类型是否为数组
	RelTypeArgIndex          = "java.rel.type_arg.index"          // 泛型参数的位置索引 (0 表示第一个参数)
	RelCastIsInstanceof      = "java.rel.cast.is_instanceof"      // 是否instanceof类型的强转
)

// --- Extended属性 (这里是定义的一些属性增强，建议按业务需求定制提取)  ---
const (
	RelCallReceiverExpression    = "java.rel.call.receiver_expression"     // 链式调用中产生接收者的表达式文本 (如 "getList()")
	RelCallIsInherited           = "java.rel.call.is_inherited"            // 调用的是否为继承自父类的方法
	RelCallTypeArguments         = "java.rel.call.type_arguments"          // 调用的泛型实参文本 (如 "String")
	RelAssignReceiver            = "java.rel.assign.receiver"              // 赋值接收者 (如 "this", "super", 或变量名)
	RelAssignIsCompound          = "java.rel.assign.is_compound"           // 是否为复合赋值 (如 +=, -=)
	RelAssignIsChained           = "java.rel.assign.is_chained"            // 是否为链式连续赋值 (如 a = b = c = 1)
	RelAssignIsPostfix           = "java.rel.assign.is_postfix"            // 是否为后置更新 (如 i++)
	RelAssignIsPrefix            = "java.rel.assign.is_prefix"             // 是否为前置更新 (如 ++i)
	RelAssignIndexExpression     = "java.rel.assign.index_expression"      // 数组赋值时的索引表达式 (如 arr[index] 中的 index)
	RelAssignIsStaticContext     = "java.rel.assign.is_static_context"     // 赋值是否发生在静态初始化块或静态方法中
	RelAssignIsParameterBinding  = "java.rel.assign.is_parameter_binding"  // 判定是否为方法入参绑定
	RelAssignIsReturnValue       = "java.rel.assign.is_return_value"       // 判定是否为方法返回值流向
	RelAssignIsCastCheck         = "java.rel.assign.is_cast_check"         // 赋值过程中是否存在类型转换
	RelAssignCastType            = "java.rel.assign.cast_type"             // 转换的目标类型
	RelAssignIsConstant          = "java.rel.assign.is_constant"           // 右值是否为字面量常量
	RelCreateIsInitializer       = "java.rel.create.is_initializer"        // 是否在声明时通过初始化器创建 (如 Field 或 Local 声明)
	RelCreateArguments           = "java.rel.create.arguments"             // 传递给构造函数的实参列表文本 (简略格式)
	RelCreateIsAnonymous         = "java.rel.create.is_anonymous"          // 是否为创建匿名内部类
	RelCreateDimensions          = "java.rel.create.dimensions"            // 数组的维度 (如 String[][] 为 2)
	RelCreateArraySize           = "java.rel.create.array_size"            // 数组创建时指定的长度文本 (如 new int[10] 中的 "10")
	RelCreateHasSubsequentCall   = "java.rel.create.has_subsequent_call"   // 创建后是否立即进行了方法调用 (new A().func())
	RelCreateSubsequentCall      = "java.rel.create.subsequent_call"       // 创建后紧跟的调用方法名
	RelCreateIsConstructorChain  = "java.rel.create.is_constructor_chain"  // 是否为构造函数链调用 (explicit super/this call)
	RelUseParentExpression       = "java.rel.use.parent_expression"        // 包含该引用的父级表达式文本 (如 "local + 2")
	RelUseUsageRole              = "java.rel.use.usage_role"               // 使用的角色 (如 "operand", "iterator_source", "array_source", "argument")
	RelUseReceiverType           = "java.rel.use.receiver_type"            // 静态字段访问的类型 QN
	RelUseIsStatic               = "java.rel.use.is_static"                // 是否为静态访问
	RelUseIndexExpression        = "java.rel.use.index_expression"         // 数组访问的索引表达式
	RelUseCallSite               = "java.rel.use.call_site"                // 作为参数传递时，被调用的方法名
	RelUseArgumentIndex          = "java.rel.use.argument_index"           // 作为参数传递时，所在的位置索引 (从 0 开始)
	RelUseContext                = "java.rel.use.context"                  // 使用的上下文语义 (如 "if_condition", "while_condition")
	RelUseTargetType             = "java.rel.use.target_type"              // 类型转换中使用时的目标类型
	RelCastOperandExpression     = "java.rel.cast.operand_expression"      // 转型操作数的原始文本 (如 "(String) obj" 中的 "obj")
	RelCastOperandKind           = "java.rel.cast.operand_kind"            // 操作数种类 (如 "variable", "method_invocation", "literal")
	RelCastIsPrimitive           = "java.rel.cast.is_primitive"            // 是否为基础数据类型的强制转换 (如 (int) double)
	RelCastTypeArguments         = "java.rel.cast.type_arguments"          // 转型目标类型的泛型参数 (如 "(List<String>)" 中的 "String")
	RelCastFullCastText          = "java.rel.cast.full_cast_text"          // 完整的转型括号文本，用于某些分析场景
	RelCastSubsequentCall        = "java.rel.cast.subsequent_call"         // 转型后紧跟的方法调用名，识别 ( (A)obj ).func()
	RelCastIsParenthesized       = "java.rel.cast.is_parenthesized"        // 是否被括号包裹 ( (Type)obj )
	RelCastIsPatternMatching     = "java.rel.cast.is_pattern_matching"     // 是否为 Java 14+ 的 instanceof 模式匹配
	RelCastPatternVariable       = "java.rel.cast.pattern_variable"        // 模式匹配中引入的新变量名 (如 "str")
	RelCastIsNestedCast          = "java.rel.cast.is_nested_cast"          // 是否为嵌套多重转型
	RelCaptureKind               = "java.rel.capture.kind"                 // 捕获的变量种类 (如 "local_variable", "parameter", "field")
	RelCaptureDepth              = "java.rel.capture.depth"                // 捕获的嵌套深度 (1表示直接外层，2及以上表示跨多层 Lambda 或内部类)
	RelCaptureIsEffectivelyFinal = "java.rel.capture.is_effectively_final" // 捕获的局部变量是否为事实上的 final
	RelCaptureIsImplicitThis     = "java.rel.capture.is_implicit_this"     // 是否通过隐式的 this 引用捕获成员变量
	RelCaptureEnclosingLambda    = "java.rel.capture.enclosing_lambda"     // 在嵌套 Lambda 场景下，记录直接包含当前 Lambda 的父 Lambda
	RelAnnotationParams          = "java.rel.annotation.params"            //
	RelAnnotationValue           = "java.rel.annotation.value"             //
	RelParameterIsFinal          = "java.rel.parameter.is_final"           // 参数是否带有 final 修饰符
	RelParameterHasAnnotation    = "java.rel.parameter.has_annotation"     // 参数是否带有注解
	RelReturnDimensions          = "java.rel.return.dimensions"            // 返回数组的维度 (如 byte[][] 为 2)
	RelReturnHasTypeArguments    = "java.rel.return.has_type_arguments"    // 返回类型是否携带泛型参数 (如 List<String>)
	RelTypeArgParentType         = "java.rel.type_arg.parent_type"         // 包含该泛型参数的主类型名 (如 List, Map)
	RelTypeArgDepth              = "java.rel.type_arg.depth"               // 嵌套深度 (0 为直接参数，1 为参数的参数，以此类推)
	RelTypeArgIsWildcard         = "java.rel.type_arg.is_wildcard"         // 是否为通配符 (如 ?, ? extends T)
	RelTypeArgWildcardKind       = "java.rel.type_arg.wildcard_kind"       // 通配符种类 (如 "extends", "super", "unbounded")
)
