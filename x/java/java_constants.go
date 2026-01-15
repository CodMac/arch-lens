package java

// 常量定义，保持元数据键名统一
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

const (
	RelRawText = "java.rel.raw_text"
	RelContext = "java.rel.ast_kind"
)
