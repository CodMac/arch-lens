package java

// =============================================================================
// java_collector 采集的 Element.Extra.Mores
// =============================================================================

const (
	BlockIsStatic                  = "java.block.is_static"           // value -> bool
	ClassIsStatic                  = "java.class.is_static"           // value -> bool
	ClassIsAbstract                = "java.class.is_abstract"         // value -> bool
	ClassIsFinal                   = "java.class.is_final"            // value -> bool
	ClassSuperClass                = "java.class.superclass"          // value -> string
	ClassImplementedInterfaces     = "java.class.interfaces"          // value -> []string
	InterfaceImplementedInterfaces = "java.interface.interfaces"      // value -> []string
	MethodIsConstructor            = "java.method.is_constructor"     // value -> bool
	MethodIsDefault                = "java.method.is_default"         // value -> bool
	MethodIsImplicit               = "java.method.is_implicit"        // value -> bool
	MethodIsAnnotation             = "java.method.is_annotation"      // value -> bool
	MethodDefaultValue             = "java.method.default_value"      // value -> string
	MethodReturnType               = "java.method.return_type"        // value -> string
	MethodParameters               = "java.method.parameters"         // value -> []string
	MethodThrowsTypes              = "java.method.throws"             // value -> []string
	VariableType                   = "java.variable.type"             // value -> string
	VariableIsFinal                = "java.variable.is_final"         // value -> bool
	VariableIsParam                = "java.variable.is_param"         // value -> bool
	FieldType                      = "java.field.type"                // value -> string
	FieldIsStatic                  = "java.field.is_static"           // value -> bool
	FieldIsFinal                   = "java.field.is_final"            // value -> bool
	FieldIsConstant                = "java.field.is_constant"         // value -> bool
	FieldIsRecordComponent         = "java.field.is_record_component" // value -> bool
	MethodRefReceiver              = "java.method_ref.receiver"       // value -> string (被引用的类或对象)
	MethodRefTarget                = "java.method_ref.target"         // value -> string (被引用的方法名)
	MethodRefTypeArgs              = "java.method_ref.type_arguments" // value -> string
	EnumArguments                  = "java.enum.arguments"            // value -> []string
	LambdaParameters               = "java.lambda.parameters"         // value -> string (参数列表字符串)
	LambdaBodyIsBlock              = "java.lambda.is_block"           // value -> bool (Body是否为代码块)
)

// =============================================================================
// java_extractor 采集的 Relation.Mores
// =============================================================================

// --- 通用属性 (Global Relations) ---
const (
	RelRawText   = "java.rel.raw_text"   // 完整的赋值语句源码 (eg,: data.name = "Hi")
	RelAstKind   = "java.rel.ast_kind"   // 最小触发单元的节点类型 (触发该关系的那个 AST 节点的类型, eg,: assignment_expression)
	RelParentAST = "java.rel.parent_ast" //
	RelContext   = "java.rel.context"    // 所属的语句块类型 (该动作发生的大环境或语句容器, eg,: expression_statement 或 method_declaration)
)

// --- CALL 关系专用 (java.rel.call) ---
const (
	RelCallReceiver           = "java.rel.call.receiver"            // 调用接收者 (如 "this", "super", 或变量名)
	RelCallReceiverType       = "java.rel.call.receiver_type"       // 接收者的类型 QN (主要用于静态调用识别类)
	RelCallReceiverExpression = "java.rel.call.receiver_expression" // 链式调用中产生接收者的表达式文本 (如 "getList()")
	RelCallIsStatic           = "java.rel.call.is_static"           // 是否为静态方法调用
	RelCallIsInherited        = "java.rel.call.is_inherited"        // 调用的是否为继承自父类的方法
	RelCallIsChained          = "java.rel.call.is_chained"          // 是否属于调用链的一部分
	RelCallIsConstructor      = "java.rel.call.is_constructor"      // 是否为构造函数调用 (new 或 explicit this/super)
	RelCallIsFunctional       = "java.rel.call.is_functional"       // 是否为方法引用 (Method Reference) 形式的调用
	RelCallTypeArguments      = "java.rel.call.type_arguments"      // 调用的泛型实参文本 (如 "String")
	RelCallEnclosingMethod    = "java.rel.call.enclosing_method"    // 调用发生的外部方法 QN (常用于 Lambda/内部类溯源)
)

// --- ASSIGN 关系专用 (java.rel.assign) ---
const (
	RelAssignOperator        = "java.rel.assign.operator"          // 赋值运算符，如 "=", "+=", "++"
	RelAssignValueExpression = "java.rel.assign.value_expression"  // 赋值语句右侧的原始表达式文本
	RelAssignTargetName      = "java.rel.assign.target_name"       // 赋值语句目标 (谁被改变了)
	RelAssignIsInitializer   = "java.rel.assign.is_initializer"    // 是否为声明时的初始化赋值 (如 int i = 0)
	RelAssignIsCompound      = "java.rel.assign.is_compound"       // 是否为复合赋值 (如 +=, -=)
	RelAssignIsChained       = "java.rel.assign.is_chained"        // 是否为链式连续赋值 (如 a = b = c = 1)
	RelAssignIsPostfix       = "java.rel.assign.is_postfix"        // 是否为后置更新 (如 i++)
	RelAssignIndexExpression = "java.rel.assign.index_expression"  // 数组赋值时的索引表达式 (如 arr[index] 中的 index)
	RelAssignIsStaticContext = "java.rel.assign.is_static_context" // 赋值是否发生在静态初始化块或静态方法中
)

// --- CREATE 关系专用 (java.rel.create) ---
const (
	RelCreateVariableName       = "java.rel.create.variable_name"        // 接收实例化对象的变量名
	RelCreateIsInitializer      = "java.rel.create.is_initializer"       // 是否在声明时通过初始化器创建 (如 Field 或 Local 声明)
	RelCreateArguments          = "java.rel.create.arguments"            // 传递给构造函数的实参列表文本 (简略格式)
	RelCreateIsAnonymous        = "java.rel.create.is_anonymous"         // 是否为创建匿名内部类
	RelCreateIsArray            = "java.rel.create.is_array"             // 是否为数组实例化
	RelCreateDimensions         = "java.rel.create.dimensions"           // 数组的维度 (如 String[][] 为 2)
	RelCreateArraySize          = "java.rel.create.array_size"           // 数组创建时指定的长度文本 (如 new int[10] 中的 "10")
	RelCreateHasSubsequentCall  = "java.rel.create.has_subsequent_call"  // 创建后是否立即进行了方法调用 (new A().func())
	RelCreateSubsequentCall     = "java.rel.create.subsequent_call"      // 创建后紧跟的调用方法名
	RelCreateIsConstructorChain = "java.rel.create.is_constructor_chain" // 是否为构造函数链调用 (explicit super/this call)
)

// --- USE 关系专用 (java.rel.use) ---
const (
	RelUseParentExpression = "java.rel.use.parent_expression" // 包含该引用的父级表达式文本 (如 "local + 2")
	RelUseUsageRole        = "java.rel.use.usage_role"        // 使用的角色 (如 "operand", "iterator_source", "array_source", "argument")
	RelUseReceiver         = "java.rel.use.receiver"          // 实例字段访问的接收者 (如 "this")
	RelUseReceiverType     = "java.rel.use.receiver_type"     // 静态字段访问的类型 QN
	RelUseIsStatic         = "java.rel.use.is_static"         // 是否为静态访问
	RelUseIndexExpression  = "java.rel.use.index_expression"  // 数组访问的索引表达式
	RelUseCallSite         = "java.rel.use.call_site"         // 作为参数传递时，被调用的方法名
	RelUseArgumentIndex    = "java.rel.use.argument_index"    // 作为参数传递时，所在的位置索引 (从 0 开始)
	RelUseContext          = "java.rel.use.context"           // 使用的上下文语义 (如 "if_condition", "while_condition")
	RelUseIsCapture        = "java.rel.use.is_capture"        // 是否为跨作用域的变量捕获引用
	RelUseEnclosingMethod  = "java.rel.use.enclosing_method"  // 捕获发生时所在的方法
	RelUseTargetType       = "java.rel.use.target_type"       // 类型转换中使用时的目标类型
)

// --- CAST 关系专用 (java.rel.cast) ---
const (
	RelCastOperandExpression = "java.rel.cast.operand_expression"  // 转型操作数的原始文本 (如 "(String) obj" 中的 "obj")
	RelCastOperandKind       = "java.rel.cast.operand_kind"        // 操作数种类 (如 "variable", "method_invocation", "literal")
	RelCastIsPrimitive       = "java.rel.cast.is_primitive"        // 是否为基础数据类型的强制转换 (如 (int) double)
	RelCastTypeArguments     = "java.rel.cast.type_arguments"      // 转型目标类型的泛型参数 (如 "(List<String>)" 中的 "String")
	RelCastFullCastText      = "java.rel.cast.full_cast_text"      // 完整的转型括号文本，用于某些分析场景
	RelCastSubsequentCall    = "java.rel.cast.subsequent_call"     // 转型后紧跟的方法调用名，识别 ( (A)obj ).func()
	RelCastIsParenthesized   = "java.rel.cast.is_parenthesized"    // 是否被括号包裹 ( (Type)obj )
	RelCastIsPatternMatching = "java.rel.cast.is_pattern_matching" // 是否为 Java 14+ 的 instanceof 模式匹配
	RelCastPatternVariable   = "java.rel.cast.pattern_variable"    // 模式匹配中引入的新变量名 (如 "str")
	RelCastIsNestedCast      = "java.rel.cast.is_nested_cast"      // 是否为嵌套多重转型
)

// --- CAPTURE 关系专用 (java.rel.capture) ---
const (
	RelCaptureKind               = "java.rel.capture.kind"                 // 捕获的变量种类 (如 "local_variable", "parameter", "field")
	RelCaptureDepth              = "java.rel.capture.depth"                // 捕获的嵌套深度 (1表示直接外层，2及以上表示跨多层 Lambda 或内部类)
	RelCaptureIsEffectivelyFinal = "java.rel.capture.is_effectively_final" // 捕获的局部变量是否为事实上的 final
	RelCaptureIsImplicitThis     = "java.rel.capture.is_implicit_this"     // 是否通过隐式的 this 引用捕获成员变量
	RelCaptureEnclosingLambda    = "java.rel.capture.enclosing_lambda"     // 在嵌套 Lambda 场景下，记录直接包含当前 Lambda 的父 Lambda
)

// --- ANNOTATION 关系专用 ---
const (
	RelAnnotationTarget = "java.rel.annotation.target"
	RelAnnotationValue  = "java.rel.annotation.value"
	RelAnnotationParams = "java.rel.annotation.params"
)

// --- THROW 关系专用 (java.rel.throw) ---
const (
	RelThrowIsSignature = "java.rel.throw.is_signature" // 是否为方法签名中 throws 关键字后的声明
	RelThrowIndex       = "java.rel.throw.index"        // 在 throws 声明列表中的位置索引 (从 0 开始)
	RelThrowIsRuntime   = "java.rel.throw.is_runtime"   // 是否为主动抛出的运行时异常 (RuntimeException 及其子类)
	RelThrowIsRethrow   = "java.rel.throw.is_rethrow"   // 是否为重新抛出已存在的异常对象 (如 throw e)
)

// --- PARAMETER 关系专用 (java.rel.parameter) ---
const (
	RelParameterName          = "java.rel.parameter.name"           // 参数定义的名称 (如 "id", "name")
	RelParameterIndex         = "java.rel.parameter.index"          // 参数在方法签名中的从 0 开始的索引位置
	RelParameterIsVarargs     = "java.rel.parameter.is_varargs"     // 是否为可变参数 (Object... args)
	RelParameterIsFinal       = "java.rel.parameter.is_final"       // 参数是否带有 final 修饰符
	RelParameterHasAnnotation = "java.rel.parameter.has_annotation" // 参数是否带有注解
)

// --- RETURN 关系专用 (java.rel.return) ---
const (
	RelReturnIsArray          = "java.rel.return.is_array"           // 返回类型是否为数组
	RelReturnDimensions       = "java.rel.return.dimensions"         // 返回数组的维度 (如 byte[][] 为 2)
	RelReturnIsPrimitive      = "java.rel.return.is_primitive"       // 返回类型是否为 Java 基础类型 (int, byte, etc.)
	RelReturnHasTypeArguments = "java.rel.return.has_type_arguments" // 返回类型是否携带泛型参数 (如 List<String>)
)

// --- TYPE_ARG 关系专用 (java.rel.type_arg) ---
const (
	RelTypeArgParentType   = "java.rel.type_arg.parent_type"   // 包含该泛型参数的主类型名 (如 List, Map)
	RelTypeArgIndex        = "java.rel.type_arg.index"         // 泛型参数的位置索引 (0 表示第一个参数)
	RelTypeArgDepth        = "java.rel.type_arg.depth"         // 嵌套深度 (0 为直接参数，1 为参数的参数，以此类推)
	RelTypeArgIsWildcard   = "java.rel.type_arg.is_wildcard"   // 是否为通配符 (如 ?, ? extends T)
	RelTypeArgWildcardKind = "java.rel.type_arg.wildcard_kind" // 通配符种类 (如 "extends", "super", "unbounded")
)
