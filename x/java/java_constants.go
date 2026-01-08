package java

const (
	// Method
	KeyIsConstructor   = "java.method.is_constructor"
	KeyFullSignatureQN = "java.method.full_signature_qn"
	KeyReturnType      = "java.method.return_type"
	KeyParameters      = "java.method.parameters"
	KeyThrowsTypes     = "java.method.throws_types"

	// Class/Interface/Enum
	KeySuperClass            = "java.class.super_class"
	KeyImplementedInterfaces = "java.class.implemented_interfaces"
	KeyIsAbstract            = "java.class.is_abstract"
	KeyIsFinal               = "java.class.is_final"

	// Field/Variable
	KeyType       = "java.field.type"
	KeyIsConstant = "java.field.is_constant"
	KeyIsParam    = "java.field.is_param"

	// EnumConstant
	KeyEnumArguments = "java.enum.arguments"
)
