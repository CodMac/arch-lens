package java

const (
	MethodIsConstructor            = "java.method.is_constructor"
	MethodFullSignatureQN          = "java.method.full_signature_qn"
	MethodReturnType               = "java.method.return_type"
	MethodParameters               = "java.method.parameters"
	MethodThrowsTypes              = "java.method.throws_types"
	ClassSuperClass                = "java.class.super_class"
	ClassImplementedInterfaces     = "java.class.implemented_interfaces"
	ClassIsAbstract                = "java.class.is_abstract"
	ClassIsFinal                   = "java.class.is_final"
	InterfaceImplementedInterfaces = "java.interface.implemented_interfaces"
	FieldType                      = "java.field.type"
	FieldIsConstant                = "java.field.is_constant" // static + final
	FieldIsFinal                   = "java.field.is_final"
	FieldIsStatic                  = "java.field.is_static"
	FieldIsParam                   = "java.field.is_param"
	VariableType                   = "java.variable.type"
	VariableIsFinal                = "java.variable.is_final"
	VariableIsParam                = "java.variable.is_param"
	EnumArguments                  = "java.enum.arguments" // USER_NOT_FOUND(404, "User not found in repository") -> (404, \"User not found in repository\")
)
