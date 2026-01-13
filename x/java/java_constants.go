package java

// 常量定义，保持元数据键名统一
const (
	BlockIsStatic                  = "java.block.is_static"
	ClassIsStatic                  = "java.class.is_static"
	ClassIsAbstract                = "java.class.is_abstract"
	ClassIsFinal                   = "java.class.is_final"
	ClassSuperClass                = "java.class.superclass"
	ClassImplementedInterfaces     = "java.class.interfaces"
	InterfaceImplementedInterfaces = "java.interface.interfaces"
	MethodIsConstructor            = "java.method.is_constructor"
	MethodIsDefault                = "java.method.is_default"
	MethodIsImplicit               = "java.method.is_implicit"
	MethodIsAnnotation             = "java.method.is_annotation"
	MethodDefaultValue             = "java.method.default_value"
	MethodReturnType               = "java.method.return_type"
	MethodParameters               = "java.method.parameters"
	MethodThrowsTypes              = "java.method.throws"
	VariableType                   = "java.variable.type"
	VariableIsFinal                = "java.variable.is_final"
	VariableIsParam                = "java.variable.is_param"
	FieldType                      = "java.field.type"
	FieldIsStatic                  = "java.field.is_static"
	FieldIsFinal                   = "java.field.is_final"
	FieldIsConstant                = "java.field.is_constant"
	FieldIsRecordComponent         = "java.field.is_record_component"
	MethodRefReceiver              = "java.method_ref.receiver" // 被引用的类或对象
	MethodRefTarget                = "java.method_ref.target"   // 被引用的方法名
	MethodRefTypeArgs              = "java.method_ref.type_arguments"
	EnumArguments                  = "java.enum.arguments"
)
