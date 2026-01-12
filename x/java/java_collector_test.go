package java_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java"

	_ "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func getTestFilePath(name string) string {
	currentDir, _ := filepath.Abs(filepath.Dir("."))
	return filepath.Join(currentDir, "testdata", name)
}

// 将返回值类型改为接口 parser.Parser
func getJavaParser(t *testing.T) parser.Parser {
	javaParser, err := parser.NewParser(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	return javaParser
}

const printEle = true

func printCodeElements(fCtx *core.FileContext) {
	if !printEle {
		return
	}

	fmt.Printf("Package: %s\n", fCtx.PackageName)
	for _, defs := range fCtx.DefinitionsBySN {
		for _, def := range defs {
			fmt.Printf("Short: %s -> Kind: %s, QN: %s\n", def.Element.Name, def.Element.Kind, def.Element.QualifiedName)
		}
	}
}

func TestJavaCollector_AbstractBaseEntity(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "AbstractBaseEntity.java"))

	// 2. 解析源码
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 3. 运行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 断言 1: 包名验证
	expectedPackage := "com.example.base"
	if fCtx.PackageName != expectedPackage {
		t.Errorf("Expected PackageName %q, got %q", expectedPackage, fCtx.PackageName)
	}

	// 断言 2: Imports 数量及内容验证
	t.Run("Verify Imports", func(t *testing.T) {
		expectedImports := []string{"java.io.Serializable", "java.util.Date"}
		if len(fCtx.Imports) != len(expectedImports) {
			t.Errorf("Expected %d imports, got %d", len(expectedImports), len(fCtx.Imports))
		}

		for _, path := range expectedImports {
			parts := strings.Split(path, ".")
			alias := parts[len(parts)-1]
			if imps, ok := fCtx.Imports[alias]; !ok || imps[0].RawImportPath != path {
				t.Errorf("Missing or incorrect import for %s", path)
			}
		}
	})

	// 断言 3: 类定义、QN、Kind、Abstract 属性、签名验证
	t.Run("Verify AbstractBaseEntity Class", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Definition not found for QN: %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		if isAbs, ok := elem.Extra.Mores[java.ClassIsAbstract].(bool); !ok || !isAbs {
			t.Error("Expected java.class.is_abstract to be true")
		}

		// 验证签名 (注意：由于 JavaCollector 内部实现可能不同，这里匹配核心部分)
		expectedSign := "public abstract class AbstractBaseEntity<ID> implements Serializable"
		if expectedSign != elem.Signature {
			t.Errorf("Signature mismatch. Got: %q, Expected: %s", elem.Signature, expectedSign)
		}
	})

	// 断言 4: 字段 id 验证
	t.Run("Verify Field id", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity.id"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field id not found")
		}

		elem := defs[0].Element
		if elem.Kind != model.Field {
			t.Errorf("Expected Field, got %s", elem.Kind)
		}

		if tpe := elem.Extra.Mores[java.FieldType]; tpe != "ID" {
			t.Errorf("Expected type ID, got %v", tpe)
		}

		if !contains(elem.Extra.Modifiers, "protected") {
			t.Error("Modifiers should contain 'protected'")
		}
	})

	// 断言 5: 字段 createdAt 验证
	t.Run("Verify Field createdAt", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity.createdAt"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field createdAt not found")
		}

		elem := defs[0].Element
		if tpe := elem.Extra.Mores[java.FieldType]; tpe != "Date" {
			t.Errorf("Expected type Date, got %v", tpe)
		}

		if !contains(elem.Extra.Modifiers, "private") {
			t.Error("Modifiers should contain 'private'")
		}
	})

	// 断言 6 & 7: 方法 getId 和 setId 验证
	t.Run("Verify Methods", func(t *testing.T) {
		// getId()
		getIdQN := "com.example.base.AbstractBaseEntity.getId()"
		getDefs := findDefinitionsByQN(fCtx, getIdQN)
		if len(getDefs) == 0 {
			t.Fatalf("Method getId() not found")
		}

		getElem := getDefs[0].Element
		if ret := getElem.Extra.Mores[java.MethodReturnType]; ret != "ID" {
			t.Errorf("getId expected return ID, got %v", ret)
		}

		// setId(ID id) - 验证 QN 括号内为类型
		setIdQN := "com.example.base.AbstractBaseEntity.setId(ID)"
		setDefs := findDefinitionsByQN(fCtx, setIdQN)
		if len(setDefs) == 0 {
			t.Fatalf("Method setId(ID) not found")
		}

		setElem := setDefs[0].Element
		if ret := setElem.Extra.Mores[java.MethodReturnType]; ret != "void" {
			t.Errorf("setId expected return void, got %v", ret)
		}
	})

	// 断言 8 & 9: 内部类 EntityMeta 及字段 tableName
	t.Run("Verify Nested Class EntityMeta", func(t *testing.T) {
		classQN := "com.example.base.AbstractBaseEntity.EntityMeta"
		classDefs := findDefinitionsByQN(fCtx, classQN)
		if len(classDefs) == 0 {
			t.Fatalf("Nested class EntityMeta not found")
		}

		classElem := classDefs[0].Element
		if !contains(classElem.Extra.Modifiers, "static") {
			t.Error("Should be static")
		}

		// 验证内部字段 tableName 的递归 QN
		fieldQN := "com.example.base.AbstractBaseEntity.EntityMeta.tableName"
		fieldDefs := findDefinitionsByQN(fCtx, fieldQN)
		if len(fieldDefs) == 0 {
			t.Fatalf("Field tableName not found in nested class")
		}

		fieldElem := fieldDefs[0].Element
		if tpe := fieldElem.Extra.Mores[java.FieldType]; tpe != "String" {
			t.Errorf("tableName expected String, got %v", tpe)
		}
	})
}

func TestJavaCollector_BaseClassHierarchy(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "BaseClass.java"))

	// 2. 解析与收集
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 断言 1 & 2: 验证 BaseClass (Abstract, Annotations, Interfaces)
	t.Run("Verify BaseClass Metadata", func(t *testing.T) {
		qn := "com.example.base.BaseClass"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("BaseClass not found")
		}
		elem := defs[0].Element

		// 断言 1: 注解验证
		expectedAnnos := []string{"@Deprecated", "@SuppressWarnings(\"unused\")"}
		for _, anno := range expectedAnnos {
			if !contains(elem.Extra.Annotations, anno) {
				t.Errorf("BaseClass missing annotation: %s", anno)
			}
		}

		// 断言 2: Abstract 属性与接口
		if isAbs, ok := elem.Extra.Mores[java.ClassIsAbstract].(bool); !ok || !isAbs {
			t.Error("Expected java.class.is_abstract to be true")
		}

		interfaces, ok := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if !ok || !contains(interfaces, "Serializable") {
			t.Errorf("Expected Serializable interface, got %v", elem.Extra.Mores[java.ClassImplementedInterfaces])
		}
	})

	// 断言 3 & 4: 验证 FinalClass (Final, SuperClass, Multiple Interfaces, Location)
	t.Run("Verify FinalClass Metadata", func(t *testing.T) {
		qn := "com.example.base.FinalClass"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("FinalClass not found")
		}
		elem := defs[0].Element

		// 断言 4: Kind 验证
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		// 断言 4: 位置信息验证 (FinalClass 在 BaseClass 之后，大致在第 11 行左右)
		if elem.Location.StartLine < 5 {
			t.Errorf("FinalClass StartLine seems incorrect: %d", elem.Location.StartLine)
		}

		// 断言 3: Final 属性
		if isFinal, ok := elem.Extra.Mores[java.ClassIsFinal].(bool); !ok || !isFinal {
			t.Error("Expected java.class.is_final to be true")
		}

		// 断言 3: 父类验证
		super, _ := elem.Extra.Mores[java.ClassSuperClass].(string)
		if !strings.Contains(super, "BaseClass") {
			t.Errorf("Expected super class BaseClass, got %q", super)
		}

		// 断言 3: 多接口验证
		interfaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if len(interfaces) < 2 || !contains(interfaces, "Cloneable") || !contains(interfaces, "Runnable") {
			t.Errorf("Expected multiple interfaces (Cloneable, Runnable), got %v", interfaces)
		}
	})

	// 断言 5: 验证 FinalClass.run() 函数的注解
	t.Run("Verify FinalClass.run() Annotations", func(t *testing.T) {
		qn := "com.example.base.FinalClass.run()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method run() not found")
		}
		elem := defs[0].Element

		if !contains(elem.Extra.Annotations, "@Override") {
			t.Error("Method run() missing @Override annotation")
		}
	})
}

func TestJavaCollector_CallbackManager(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "CallbackManager.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 验证 1: 验证方法内部定义的局部类 LocalValidator
	t.Run("Verify Local Class", func(t *testing.T) {
		// 根据你的 Collector 实现，局部类应该在方法 QN 下
		qn := "com.example.base.CallbackManager.register().LocalValidator"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Local class LocalValidator not found at %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		// 验证局部类内部的方法
		methodQN := qn + ".isValid()"
		methodDefs := findDefinitionsByQN(fCtx, methodQN)
		if len(methodDefs) == 0 {
			t.Errorf("Method isValid() not found in local class")
		}
		if methodDefs[0].Element.Extra.Mores[java.MethodParameters] != nil {
			t.Errorf("Method isValid() found params")
		}
	})

	// 验证 2: 验证变量 r
	t.Run("Verify Variable r", func(t *testing.T) {
		qn := "com.example.base.CallbackManager.register().r"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable r not found at %s", qn)
		}

		elem := defs[0].Element
		if tpe := elem.Extra.Mores[java.VariableType]; tpe != "Runnable" {
			t.Errorf("Expected type Runnable, got %v", tpe)
		}
	})

	// 验证 3: 验证匿名内部类及其方法 run()
	t.Run("Verify Anonymous Inner Class and Run Method", func(t *testing.T) {
		// 修正路径：anonymousClass$1 现在应该正确嵌套了 run()
		anonQN := "com.example.base.CallbackManager.register().anonymousClass$1"
		runQN := anonQN + ".run()"

		runDefs := findDefinitionsByQN(fCtx, runQN)
		if len(runDefs) == 0 {
			t.Fatalf("Method run() not found at expected QN: %s", runQN)
		}

		elem := runDefs[0].Element
		if !contains(elem.Extra.Annotations, "@Override") {
			t.Error("Method run() missing @Override")
		}
	})
}

func TestJavaCollector_ConfigService(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "ConfigService.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 验证 1: 变长参数 (Object...) 与 数组参数 (String[])
	t.Run("Verify Variadic and Array Parameters", func(t *testing.T) {
		// 注意：QN 内部的参数类型应反映原始定义
		qn := "com.example.base.ConfigService.updateConfigs(String[],Object...)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method updateConfigs not found with expected signature QN: %s", qn)
		}

		elem := defs[0].Element
		params, ok := elem.Extra.Mores[java.MethodParameters].([]string)
		if !ok || len(params) != 2 {
			t.Fatalf("Expected 2 parameters, got %v", params)
		}

		// 验证数组参数
		if !strings.Contains(params[0], "String[]") {
			t.Errorf("Expected first param to be String[], got %s", params[0])
		}

		// 验证变长参数
		if !strings.Contains(params[1], "Object...") {
			t.Errorf("Expected second param to be Object..., got %s", params[1])
		}
	})

	// 验证 2: 复杂多属性注解
	t.Run("Verify Complex Annotations", func(t *testing.T) {
		qn := "com.example.base.ConfigService.legacyMethod()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method legacyMethod not found")
		}

		elem := defs[0].Element
		annos := elem.Extra.Annotations

		// 验证 @SuppressWarnings 的数组格式
		foundSuppressed := false
		for _, a := range annos {
			if strings.Contains(a, "@SuppressWarnings") && strings.Contains(a, "\"unchecked\"") && strings.Contains(a, "\"rawtypes\"") {
				foundSuppressed = true
				break
			}
		}
		if !foundSuppressed {
			t.Errorf("Could not find complete @SuppressWarnings annotation, got: %v", annos)
		}

		// 验证 @Deprecated 的多属性 (since, forRemoval)
		foundDeprecated := false
		for _, a := range annos {
			if strings.Contains(a, "@Deprecated") && strings.Contains(a, "since = \"1.2\"") && strings.Contains(a, "forRemoval = true") {
				foundDeprecated = true
				break
			}
		}
		if !foundDeprecated {
			t.Errorf("Could not find detailed @Deprecated annotation, got: %v", annos)
		}
	})

	t.Run("Verify Specific Parameters", func(t *testing.T) {
		// 验证 keys
		keysQN := "com.example.base.ConfigService.updateConfigs(String[],Object...).keys"
		if len(findDefinitionsByQN(fCtx, keysQN)) == 0 {
			t.Errorf("Variable 'keys' not found")
		}

		// 验证 values
		valuesQN := "com.example.base.ConfigService.updateConfigs(String[],Object...).values"
		vDefs := findDefinitionsByQN(fCtx, valuesQN)
		if len(vDefs) == 0 {
			t.Fatalf("Variable 'values' not found")
		}

		vElem := vDefs[0].Element
		if tpe := vElem.Extra.Mores[java.VariableType]; tpe != "Object..." {
			t.Errorf("Expected type Object..., got %v", tpe)
		}
	})
}

func TestJavaCollector_DataProcessor(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "DataProcessor.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 验证 1: 接口定义、多继承与泛型
	t.Run("Verify Interface Heritage and Generics", func(t *testing.T) {
		qn := "com.example.base.DataProcessor"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Interface DataProcessor not found")
		}

		elem := defs[0].Element
		// 验证接口继承
		ifaces, _ := elem.Extra.Mores[java.InterfaceImplementedInterfaces].([]string)
		expectedIfaces := []string{"Runnable", "AutoCloseable"}
		for _, expected := range expectedIfaces {
			if !contains(ifaces, expected) {
				t.Errorf("Expected interface %s not found in %v", expected, ifaces)
			}
		}

		// 验证签名中的泛型参数 (T extends AbstractBaseEntity<?>)
		if !strings.Contains(elem.Signature, "<T extends AbstractBaseEntity<?>>") {
			t.Errorf("Signature missing generics: %s", elem.Signature)
		}
	})

	// 验证 2: 方法的 Throws 异常
	t.Run("Verify Method Throws", func(t *testing.T) {
		// 注意：泛型 T 在 QN 中按原样提取
		qn := "com.example.base.DataProcessor.processAll(String)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method processAll not found")
		}

		elem := defs[0].Element
		throws, _ := elem.Extra.Mores[java.MethodThrowsTypes].([]string)

		expectedThrows := []string{"RuntimeException", "Exception"}
		if len(throws) != 2 {
			t.Fatalf("Expected 2 throws types, got %v", throws)
		}
		for i, e := range expectedThrows {
			if throws[i] != e {
				t.Errorf("Expected throw %s, got %s", e, throws[i])
			}
		}
	})

	// 验证 3: Java 8 Default 方法修饰符
	t.Run("Verify Default Method", func(t *testing.T) {
		qn := "com.example.base.DataProcessor.stop()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method stop not found")
		}

		elem := defs[0].Element
		// 验证是否包含 default 关键字
		if !contains(elem.Extra.Modifiers, "default") {
			t.Errorf("Method stop should have 'default' modifier, got %v", elem.Extra.Modifiers)
		}

		// 验证 Signature 是否正确包含 default
		if !strings.HasPrefix(elem.Signature, "default void stop()") {
			t.Errorf("Signature prefix incorrect: %s", elem.Signature)
		}
	})
}

func TestJavaCollector_NestedAndStaticBlocks(t *testing.T) {
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "OuterClass.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 验证 1: 静态初始化块与实例块
	t.Run("Verify Initialization Blocks", func(t *testing.T) {
		// 静态块通常被识别为 static_initializer 节点, 我们将其命名为 $static
		staticBlockQN := "com.example.base.OuterClass.$static$1"
		if len(findDefinitionsByQN(fCtx, staticBlockQN)) == 0 {
			t.Errorf("Static initializer block not found at expected QN: %s", staticBlockQN)
		}
	})

	// 验证 2: 内部类与静态嵌套类
	t.Run("Verify Nested Classes", func(t *testing.T) {
		// 内部类 QN
		innerQN := "com.example.base.OuterClass.InnerClass"
		if len(findDefinitionsByQN(fCtx, innerQN)) == 0 {
			t.Errorf("InnerClass not found")
		}

		// 静态嵌套类方法 QN
		nestedMethodQN := "com.example.base.OuterClass.StaticNestedClass.run()"
		if len(findDefinitionsByQN(fCtx, nestedMethodQN)) == 0 {
			t.Errorf("Method run() in StaticNestedClass not found")
		}
	})

	// 验证 3: 方法内部类 (Local Class)
	t.Run("Verify Local Class", func(t *testing.T) {
		// 注意层级：OuterClass -> scopeTest() -> LocalClass
		localClassQN := "com.example.base.OuterClass.scopeTest().LocalClass"
		defs := findDefinitionsByQN(fCtx, localClassQN)
		if len(defs) == 0 {
			t.Errorf("Local class inside method not found at: %s", localClassQN)
		}
	})
}

func TestJavaCollector_Annotation(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "annotation", "Loggable.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, true, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 验证 1: Annotation Type Declaration & 注释提取
	t.Run("Verify Annotation Declaration and Doc", func(t *testing.T) {
		qn := "com.example.base.annotation.Loggable"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Annotation Loggable not found with QN: %s", qn)
		}

		elem := defs[0].Element
		// 验证注释提取 (Doc)
		if !strings.Contains(elem.Doc, "Annotation Type Declaration") || !strings.Contains(elem.Doc, "Meta-Annotations") {
			t.Errorf("Doc comment not correctly extracted, got: %s", elem.Doc)
		}

		// 验证元注解 (Meta-Annotations)
		annos := elem.Extra.Annotations
		hasRetention := false
		hasTarget := false
		for _, a := range annos {
			if strings.Contains(a, "@Retention") {
				hasRetention = true
			}
			if strings.Contains(a, "@Target") {
				hasTarget = true
			}
		}
		if !hasRetention || !hasTarget {
			t.Errorf("Missing meta-annotations. Found: %v", annos)
		}
	})

	// 验证 2: 语义化 Import ("*" 通配符)
	t.Run("Verify Wildcard Import", func(t *testing.T) {
		// 在 map[string][]*ImportEntry 中，通配符导入的 key 通常是 "*"
		imports, ok := fCtx.Imports["*"]
		if !ok || len(imports) == 0 {
			t.Fatalf("Wildcard imports not found in FileContext")
		}

		found := false
		for _, imp := range imports {
			if imp.RawImportPath == "java.lang.annotation.*" && imp.IsWildcard {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected wildcard import 'java.lang.annotation.*' not found under key '*'")
		}
	})

	// 验证 3: 注解的函数定义、默认返回值及特殊属性
	t.Run("Verify Annotation Members", func(t *testing.T) {
		// 调整：注解成员在 QN 中不带括号，因为它们不是真正的 method_declaration
		levelQN := "com.example.base.annotation.Loggable.level()"
		levelDefs := findDefinitionsByQN(fCtx, levelQN)
		if len(levelDefs) == 0 {
			t.Fatalf("Annotation member level not found with QN: %s", levelQN)
		}

		levelElem := levelDefs[0].Element
		if isAnno, _ := levelElem.Extra.Mores[java.MethodIsAnnotation].(bool); !isAnno {
			t.Errorf("level should have MethodIsAnnotation = true")
		}
		if defVal := levelElem.Extra.Mores[java.MethodDefaultValue]; defVal != "\"INFO\"" {
			t.Errorf("Expected default value \"INFO\", got %v", defVal)
		}

		// 验证 trace 及其默认值
		traceQN := "com.example.base.annotation.Loggable.trace()"
		traceDefs := findDefinitionsByQN(fCtx, traceQN)
		if len(traceDefs) == 0 {
			t.Fatalf("Annotation member trace not found")
		}

		traceElem := traceDefs[0].Element
		if defVal := traceElem.Extra.Mores[java.MethodDefaultValue]; defVal != "false" {
			t.Errorf("Expected default value false, got %v", traceElem.Extra.Mores[java.MethodDefaultValue])
		}
	})
}

func TestJavaCollector_EnumErrorCode(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "enum", "ErrorCode.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证枚举主体及其全限定名 (QN)
	t.Run("Verify Enum Entity", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum ErrorCode not found")
		}
		elem := defs[0].Element
		if elem.Kind != model.Enum {
			t.Errorf("Expected Kind ENUM, got %s", elem.Kind)
		}
	})

	// 2. 验证枚举常量及其参数 (使用 java.EnumArguments)
	t.Run("Verify Enum Constant Arguments", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode.USER_NOT_FOUND"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum constant USER_NOT_FOUND not found")
		}

		elem := defs[0].Element
		// 验证枚举常量被识别为 Field (对应你的 identifyElement 逻辑)
		if elem.Kind != model.EnumConstant {
			t.Errorf("Expected Enum Constant to be Kind Field, got %s", elem.Kind)
		}

		// 核心验证：检查参数提取 (404, "User not found...")
		args, ok := elem.Extra.Mores[java.EnumArguments].([]string)
		if !ok {
			t.Fatalf("Metadata key %s (EnumArguments) not found or wrong type", java.EnumArguments)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 arguments, got %d", len(args))
		}
		if args[0] != "404" {
			t.Errorf("Expected first arg 404, got %s", args[0])
		}
	})

	// 3. 验证构造函数 (使用 java.MethodIsConstructor)
	t.Run("Verify Enum Constructor", func(t *testing.T) {
		// 构造函数 QN 包含参数类型
		qn := "com.example.base.enum.ErrorCode.ErrorCode(int,String)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum constructor with QN %s not found", qn)
		}

		elem := defs[0].Element
		isCtor, ok := elem.Extra.Mores[java.MethodIsConstructor].(bool)
		if !ok || !isCtor {
			t.Errorf("Expected %s to be true", java.MethodIsConstructor)
		}
	})

	// 4. 验证成员方法及其返回值类型 (使用 java.MethodReturnType)
	t.Run("Verify Enum Member Methods", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode.getMessage()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method getMessage() not found")
		}

		elem := defs[0].Element
		retType, ok := elem.Extra.Mores[java.MethodReturnType].(string)
		if !ok || retType != "String" {
			t.Errorf("Expected return type String, got %v", retType)
		}
	})
}

func TestJavaCollector_NotificationException(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "exception", "NotificationException.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证类的继承关系 (EXTEND)
	t.Run("Verify Exception Inheritance", func(t *testing.T) {
		qn := "com.example.base.exception.NotificationException"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Class NotificationException not found")
		}
		elem := defs[0].Element

		// 验证 SuperClass 字段 (对应 java.class.superclass)
		super, ok := elem.Extra.Mores[java.ClassSuperClass].(string)
		if !ok || super != "Exception" {
			t.Errorf("Expected superclass 'Exception', got '%v'", super)
		}
	})

	// 2. 验证序列化常量 (Field)
	t.Run("Verify serialVersionUID Field", func(t *testing.T) {
		qn := "com.example.base.exception.NotificationException.serialVersionUID"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field serialVersionUID not found")
		}
		elem := defs[0].Element

		// 验证常量属性 (static + final)
		isConstant := elem.Extra.Mores[java.FieldIsConstant].(bool)
		if !isConstant {
			t.Error("serialVersionUID should be identified as a constant")
		}

		fieldType := elem.Extra.Mores[java.FieldType].(string)
		if fieldType != "long" {
			t.Errorf("Expected type long, got %s", fieldType)
		}
	})

	// 3. 验证多个构造函数 (Constructor Overloading)
	t.Run("Verify Overloaded Constructors", func(t *testing.T) {
		// 构造函数 A: (String, Throwable)
		qnA := "com.example.base.exception.NotificationException.NotificationException(String,Throwable)"
		defsA := findDefinitionsByQN(fCtx, qnA)
		if len(defsA) == 0 {
			t.Fatalf("Constructor (String, Throwable) not found")
		}
		if !defsA[0].Element.Extra.Mores[java.MethodIsConstructor].(bool) {
			t.Error("Should be marked as constructor")
		}

		// 构造函数 B: (ErrorCode)
		qnB := "com.example.base.exception.NotificationException.NotificationException(ErrorCode)"
		defsB := findDefinitionsByQN(fCtx, qnB)
		if len(defsB) == 0 {
			t.Fatalf("Constructor (ErrorCode) not found")
		}

		// 验证参数元数据
		params, _ := defsB[0].Element.Extra.Mores[java.MethodParameters].([]string)
		if len(params) != 1 || !strings.Contains(params[0], "ErrorCode code") {
			t.Errorf("Incorrect parameters metadata: %v", params)
		}
	})
}

func TestJavaCollector_User(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "User.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证静态导入 (Static Imports)
	t.Run("Verify Static Imports", func(t *testing.T) {
		// 静态导入 DAYS
		if imp, ok := fCtx.Imports["DAYS"]; !ok {
			t.Error("Static import 'DAYS' not found in FileContext")
		} else {
			entry := imp[0]
			if entry.Kind != model.Constant {
				t.Errorf("Expected DAYS to be Kind Constant, got %s", entry.Kind)
			}
			if entry.RawImportPath != "java.util.concurrent.TimeUnit.DAYS" {
				t.Errorf("Incorrect path for DAYS: %s", entry.RawImportPath)
			}
		}
	})

	// 2. 验证静态常量 (Static Final Field)
	t.Run("Verify Constant Field DEFAULT_ID", func(t *testing.T) {
		qn := "com.example.base.User.DEFAULT_ID"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field DEFAULT_ID not found")
		}
		elem := defs[0].Element

		// 验证元数据中的常量标记
		if isConst, _ := elem.Extra.Mores[java.FieldIsConstant].(bool); !isConst {
			t.Error("DEFAULT_ID should be identified as a Constant (static + final)")
		}
		if fType := elem.Extra.Mores[java.FieldType].(string); fType != "String" {
			t.Errorf("Expected field type String, got %s", fType)
		}
	})

	// 3. 验证静态内部类 (Nested Class)
	t.Run("Verify Inner Class AddonInfo", func(t *testing.T) {
		qn := "com.example.base.User.AddonInfo"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class AddonInfo not found")
		}
		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind Class, got %s", elem.Kind)
		}

		// 验证内部类字段的递归 QN
		fieldQN := "com.example.base.User.AddonInfo.otherName"
		if len(findDefinitionsByQN(fCtx, fieldQN)) == 0 {
			t.Errorf("Field otherName in inner class not found with QN: %s", fieldQN)
		}
	})

	// 4. 验证 if 块产生的作用域 (ScopeBlock)
	t.Run("Verify If Blocks in chooseUnit", func(t *testing.T) {
		// chooseUnit 方法内部有多个 if 块。
		// 根据你的 applyUniqueQN 逻辑，它们应该被命名为 block$1, block$2 等
		methodQN := "com.example.base.User.AddonInfo.chooseUnit(long)"

		// 验证 block$1 (第一个 if 分支的内容)
		block1QN := methodQN + ".block$1"
		defs := findDefinitionsByQN(fCtx, block1QN)
		if len(defs) == 0 {
			t.Fatalf("First if-block (block$1) not found in chooseUnit")
		}

		elem := defs[0].Element
		if elem.Kind != model.ScopeBlock {
			t.Errorf("Expected ScopeBlock, got %s", elem.Kind)
		}

		// 验证 block$2 (第二个 if 分支)
		block2QN := methodQN + ".block$2"
		if len(findDefinitionsByQN(fCtx, block2QN)) == 0 {
			t.Error("Second if-block (block$2) not found")
		}
	})
}

func TestJavaCollector_UserServiceImpl(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "UserServiceImpl.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证类声明：包含注解、复杂的泛型继承与实现
	t.Run("Verify UserServiceImpl Class Definition", func(t *testing.T) {
		qn := "com.example.base.UserServiceImpl"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Class UserServiceImpl not found")
		}
		elem := defs[0].Element

		// 验证注解
		if !contains(elem.Extra.Annotations, "@Loggable") {
			t.Errorf("Expected annotation @Loggable, got %v", elem.Extra.Annotations)
		}

		// 验证泛型父类
		if super := elem.Extra.Mores[java.ClassSuperClass].(string); super != "AbstractBaseEntity<String>" {
			t.Errorf("Expected superclass AbstractBaseEntity<String>, got %s", super)
		}

		// 验证泛型接口实现 (DataProcessor<AbstractBaseEntity<String>>)
		ifaces, ok := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if !ok || !contains(ifaces, "DataProcessor<AbstractBaseEntity<String>>") {
			t.Errorf("Expected interface DataProcessor<AbstractBaseEntity<String>> in %v", ifaces)
		}

		// 验证完整 Signature
		expectedSig := "public class UserServiceImpl extends AbstractBaseEntity<String> implements DataProcessor<AbstractBaseEntity<String>>"
		if elem.Signature != expectedSig {
			t.Errorf("Signature mismatch.\nGot: %s\nExp: %s", elem.Signature, expectedSig)
		}
	})

	// 2. 验证泛型方法：包含 Override 注解、泛型返回值、Throws 异常
	t.Run("Verify Method processAll", func(t *testing.T) {
		qn := "com.example.base.UserServiceImpl.processAll(String)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method processAll(String) not found")
		}
		elem := defs[0].Element

		// 验证泛型返回值
		if ret := elem.Extra.Mores[java.MethodReturnType].(string); ret != "List<AbstractBaseEntity<String>>" {
			t.Errorf("Expected return type List<AbstractBaseEntity<String>>, got %s", ret)
		}

		// 验证 Throws 声明
		throws, ok := elem.Extra.Mores[java.MethodThrowsTypes].([]string)
		if !ok || !contains(throws, "RuntimeException") {
			t.Errorf("Expected throws RuntimeException, got %v", throws)
		}

		// 验证方法 Signature (应包含 public 和 throws)
		if !strings.Contains(elem.Signature, "public List<AbstractBaseEntity<String>> processAll(String batchId)") {
			t.Errorf("Signature should contain access modifier and generic return type, got: %s", elem.Signature)
		}
		if !strings.Contains(elem.Signature, "throws RuntimeException") {
			t.Errorf("Signature should contain throws clause, got: %s", elem.Signature)
		}
	})

	// 3. 验证方法体内的局部变量 (Local Variables)
	t.Run("Verify Local Variables in processAll", func(t *testing.T) {
		methodQN := "com.example.base.UserServiceImpl.processAll(String)"

		// 验证 results 变量
		resultsQN := methodQN + ".results"
		rDefs := findDefinitionsByQN(fCtx, resultsQN)
		if len(rDefs) == 0 {
			t.Errorf("Local variable 'results' not found with QN: %s", resultsQN)
		} else {
			vType := rDefs[0].Element.Extra.Mores[java.VariableType].(string)
			if vType != "List<AbstractBaseEntity<String>>" {
				t.Errorf("Incorrect type for results: %s", vType)
			}
		}

		// 验证 converted 变量 (Cast 表达式后的变量)
		convertedQN := methodQN + ".converted"
		cDefs := findDefinitionsByQN(fCtx, convertedQN)
		if len(cDefs) == 0 {
			t.Errorf("Local variable 'converted' not found")
		} else {
			vType := cDefs[0].Element.Extra.Mores[java.VariableType].(string)
			if vType != "String" {
				t.Errorf("Expected type String for 'converted', got %s", vType)
			}
		}
	})

	// 4. 验证构造函数及其内的 Field Access (隐式验证 QN 深度)
	t.Run("Verify Constructor and Implicit Logic", func(t *testing.T) {
		// 构造函数 QN 通常以类名命名
		ctorQN := "com.example.base.UserServiceImpl.UserServiceImpl()"
		defs := findDefinitionsByQN(fCtx, ctorQN)
		if len(defs) == 0 {
			t.Fatalf("Constructor UserServiceImpl() not found")
		}

		elem := defs[0].Element
		if !elem.Extra.Mores[java.MethodIsConstructor].(bool) {
			t.Error("Should be identified as a constructor")
		}
	})
}

func TestJavaCollector_AnonymousAndNested(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "AnonymousClassTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证匿名内部类 (通常 QN 带有 $1 等后缀)
	t.Run("Verify Anonymous Class", func(t *testing.T) {
		// 匿名类位于 run 方法内部
		qn := "com.example.base.test.AnonymousClassTest.run().anonymousClass$1"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Anonymous class not found at %s", qn)
		}
		if defs[0].Element.Kind != model.AnonymousClass {
			t.Errorf("Expected AnonymousClass, got %s", defs[0].Element.Kind)
		}

		// 验证匿名内部类里的方法
		methodQN := qn + ".compareTo(String)"
		if len(findDefinitionsByQN(fCtx, methodQN)) == 0 {
			t.Errorf("Method compareTo not found inside anonymous class")
		}
	})

	// 2. 验证静态内部类
	t.Run("Verify Static Inner Class", func(t *testing.T) {
		qn := "com.example.base.test.AnonymousClassTest.Inner"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class not found")
		}

		isStatic := defs[0].Element.Extra.Mores[java.ClassIsStatic].(bool)
		if !isStatic {
			t.Errorf("Expected Inner class to be static")
		}
	})

	// 3. 验证嵌套在内部类里的枚举
	t.Run("Verify Nested Enum in Inner Class", func(t *testing.T) {
		qn := "com.example.base.test.AnonymousClassTest.Inner.Color"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Nested enum Color not found")
		}
		if defs[0].Element.Kind != model.Enum {
			t.Errorf("Expected Enum, got %s", defs[0].Element.Kind)
		}

		// 验证枚举项
		constantQN := qn + ".RED"
		if len(findDefinitionsByQN(fCtx, constantQN)) == 0 {
			t.Errorf("Enum constant RED not found")
		}
	})
}

func TestJavaCollector_ExtendAndImplement(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "ExtendAndImplementTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证抽象类 BaseClass
	t.Run("Verify Abstract BaseClass", func(t *testing.T) {
		qn := "com.example.base.test.BaseClass"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("BaseClass not found")
		}
		elem := defs[0].Element

		// 验证修饰符
		if isAbs := elem.Extra.Mores[java.ClassIsAbstract].(bool); !isAbs {
			t.Errorf("Expected ClassIsAbstract to be true")
		}

		// 验证接口实现
		ifaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if len(ifaces) != 1 || ifaces[0] != "Serializable" {
			t.Errorf("Expected interface Serializable, got %v", ifaces)
		}

		// 验证注解 (存在两个注解)
		annos := elem.Extra.Annotations
		if len(annos) != 2 {
			t.Errorf("Expected 2 annotations, got %d", len(annos))
		}
	})

	// 2. 验证最终类 FinalClass
	t.Run("Verify Final FinalClass", func(t *testing.T) {
		qn := "com.example.base.test.FinalClass"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("FinalClass not found")
		}
		elem := defs[0].Element

		// 验证 Final 状态
		if isFinal := elem.Extra.Mores[java.ClassIsFinal].(bool); !isFinal {
			t.Errorf("Expected ClassIsFinal to be true")
		}

		// 验证父类继承
		super, _ := elem.Extra.Mores[java.ClassSuperClass].(string)
		if super != "BaseClass" {
			t.Errorf("Expected SuperClass BaseClass, got %s", super)
		}

		// 验证多接口实现
		ifaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		expectedIfaces := []string{"Cloneable", "Runnable"}
		if len(ifaces) != 2 {
			t.Errorf("Expected 2 interfaces, got %v", ifaces)
		}
		for _, expected := range expectedIfaces {
			found := false
			for _, got := range ifaces {
				if got == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected interface %s not found in %v", expected, ifaces)
			}
		}
	})

	// 3. 验证方法重写与 Signature
	t.Run("Verify Override Method Signature", func(t *testing.T) {
		qn := "com.example.base.test.FinalClass.run()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method run() not found")
		}
		elem := defs[0].Element

		// 验证包含 @Override 注解
		hasOverride := false
		for _, anno := range elem.Extra.Annotations {
			if strings.Contains(anno, "Override") {
				hasOverride = true
				break
			}
		}
		if !hasOverride {
			t.Errorf("Method run() should have @Override annotation")
		}
	})
}

func TestJavaCollector_GenericComplex(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "GenericTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证接口泛型边界 (Intersection Type: Serializable & Cloneable)
	t.Run("Verify Interface Generic Bounds", func(t *testing.T) {
		qn := "com.example.base.test.GenericTest"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Interface GenericTest not found")
		}
		elem := defs[0].Element

		// 验证 Signature 是否保留了泛型边界
		// 预期: interface GenericTest<T extends Serializable & Cloneable>
		if !strings.Contains(elem.Signature, "<T extends Serializable & Cloneable>") {
			t.Errorf("Signature missing intersection type bounds: %s", elem.Signature)
		}
	})

	// 2. 验证复杂的方法参数与返回类型 (List<? extends T>)
	t.Run("Verify Complex Wildcard Method", func(t *testing.T) {
		// 注意：QN 构建时会提取参数类型，并移除泛型部分以保证稳定性
		qn := "com.example.base.test.GenericTest.findAllByCriteria(List)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method findAllByCriteria not found")
		}
		elem := defs[0].Element

		// 验证返回值 (MethodReturnType)
		retType, _ := elem.Extra.Mores[java.MethodReturnType].(string)
		if retType != "List<? extends T>" {
			t.Errorf("Expected return type List<? extends T>, got %s", retType)
		}

		// 验证原始参数列表 (MethodParameters)
		params, _ := elem.Extra.Mores[java.MethodParameters].([]string)
		if len(params) == 0 || params[0] != "List<? super T> criteria" {
			t.Errorf("Expected param List<? super T> criteria, got %v", params)
		}
	})

	// 3. 验证方法级泛型与异常声明 (throws E)
	t.Run("Verify Method-level Generics and Throws", func(t *testing.T) {
		qn := "com.example.base.test.GenericTest.executeOrThrow(E)"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method executeOrThrow not found")
		}
		elem := defs[0].Element

		// 验证 Signature 包含泛型定义 <E extends Exception>
		if !strings.Contains(elem.Signature, "<E extends Exception>") {
			t.Errorf("Signature missing method-level generic: %s", elem.Signature)
		}

		// 验证 Throws 元数据
		throws, _ := elem.Extra.Mores[java.MethodThrowsTypes].([]string)
		if len(throws) == 0 || throws[0] != "E" {
			t.Errorf("Expected throws E, got %v", throws)
		}
	})
}

func TestJavaCollector_Imports(t *testing.T) {
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "ImportTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// 1. 验证 Class 导入: import java.util.List;
	t.Run("Verify Specific Class Import", func(t *testing.T) {
		imps, exists := fCtx.Imports["List"]
		if !exists || len(imps) == 0 {
			t.Fatalf("Import 'List' not found")
		}
		// 取第一个（通常 List 在此类中只有一个导入）
		imp := imps[0]
		if imp.RawImportPath != "java.util.List" {
			t.Errorf("Expected java.util.List, got %s", imp.RawImportPath)
		}
		if imp.Kind != model.Class {
			t.Errorf("Expected Kind Class, got %s", imp.Kind)
		}
	})

	// 2. 验证通配符导入 (重点：验证 slice 中包含两个通配符)
	t.Run("Verify Multiple Wildcard Imports", func(t *testing.T) {
		imps, exists := fCtx.Imports["*"]
		if !exists || len(imps) < 2 {
			t.Fatalf("Expected at least 2 wildcard imports, found %d", len(imps))
		}

		paths := make(map[string]bool)
		for _, imp := range imps {
			paths[imp.RawImportPath] = true
		}

		// 验证 java.util.*
		if !paths["java.util.*"] {
			t.Errorf("Wildcard import java.util.* not found in %v", paths)
		}
		// 验证 static java.lang.Math.*
		if !paths["java.lang.Math.*"] {
			t.Errorf("Static wildcard import java.lang.Math.* not found in %v", paths)
		}
	})

	// 3. 验证具体静态方法导入: import static java.util.Collections.sort;
	t.Run("Verify Static Method Import", func(t *testing.T) {
		imps, exists := fCtx.Imports["sort"]
		if !exists || len(imps) == 0 {
			t.Fatalf("Static import 'sort' not found")
		}
		imp := imps[0]
		if imp.RawImportPath != "java.util.Collections.sort" {
			t.Errorf("Expected java.util.Collections.sort, got %s", imp.RawImportPath)
		}
		// 静态非通配符导入，handleImport 标记为 Constant
		if imp.Kind != model.Constant {
			t.Errorf("Expected Kind Constant, got %s", imp.Kind)
		}
	})
}

func TestJavaCollector_MethodOverloading(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "MethodTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证构造函数识别
	t.Run("Verify Constructor", func(t *testing.T) {
		qn := "com.example.base.MethodTest.MethodTest()"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Constructor not found by QN: %s", qn)
		}
		elem := defs[0].Element

		if isCons, _ := elem.Extra.Mores[java.MethodIsConstructor].(bool); !isCons {
			t.Errorf("Expected MethodIsConstructor to be true")
		}
		if elem.Kind != model.Method {
			t.Errorf("Expected Kind Method, got %s", elem.Kind)
		}
	})

	// 2. 验证方法重载 (Overloading) 的 QN 唯一性
	t.Run("Verify Method Overloading QNs", func(t *testing.T) {
		overloads := []struct {
			qn       string
			expected string
		}{
			{"com.example.base.MethodTest.exec(int)", "void"},
			{"com.example.base.MethodTest.exec(String)", "String"},
			{"com.example.base.MethodTest.exec(int,String)", "void"},
		}

		for _, tc := range overloads {
			defs := findDefinitionsByQN(fCtx, tc.qn)
			if len(defs) == 0 {
				t.Errorf("Method overload not found: %s", tc.qn)
				continue
			}

			// 验证返回值提取
			retType, _ := defs[0].Element.Extra.Mores[java.MethodReturnType].(string)
			if retType != tc.expected {
				t.Errorf("For %s, expected return type %s, got %s", tc.qn, tc.expected, retType)
			}
		}
	})

	// 3. 验证 FileContext 里的 SN (Short Name) 聚合
	t.Run("Verify SN Aggregation", func(t *testing.T) {
		// exec 这个名字应该对应 3 个定义
		entries, exists := fCtx.DefinitionsBySN["exec"]
		if !exists {
			t.Fatalf("SN 'exec' not found in DefinitionsBySN")
		}
		if len(entries) != 3 {
			t.Errorf("Expected 3 overloads for 'exec', found %d", len(entries))
		}

		// 验证 QN 是否各不相同
		qnSet := make(map[string]bool)
		for _, entry := range entries {
			qnSet[entry.Element.QualifiedName] = true
		}
		if len(qnSet) != 3 {
			t.Errorf("Duplicate QNs detected in SN aggregation: %v", qnSet)
		}
	})
}

func TestJavaCollector_ParameterScope(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "ParameterScopeTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证构造函数参数
	t.Run("Verify Constructor Parameter", func(t *testing.T) {
		// 注意 QN 路径：类 -> 构造函数 -> 参数
		qn := "com.example.base.test.ParameterScopeTest.ParameterScopeTester(String).initialConfig"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Constructor parameter 'initialConfig' not found at %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Variable {
			t.Errorf("Expected Kind Variable, got %s", elem.Kind)
		}
		if isParam := elem.Extra.Mores[java.VariableIsParam].(bool); !isParam {
			t.Error("Expected VariableIsParam to be true")
		}
	})

	// 2. 验证变长参数 (Varargs)
	t.Run("Verify Varargs Parameter", func(t *testing.T) {
		qn := "com.example.base.test.ParameterScopeTest.execute(int,String...).labels"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Varargs parameter 'labels' not found")
		}

		vType := defs[0].Element.Extra.Mores[java.VariableType].(string)
		// 验证你的 extractTypeString 是否正确处理了 "..."
		if !strings.Contains(vType, "...") {
			t.Errorf("Expected type with '...', got %s", vType)
		}
	})

	// 3. 验证内部类方法参数的作用域层级
	t.Run("Verify Inner Class Method Parameter", func(t *testing.T) {
		qn := "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long).duration"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class method parameter 'duration' not found")
		}

		// 验证路径层级是否包含 InnerWorker
		if !strings.Contains(defs[0].Element.QualifiedName, "InnerWorker") {
			t.Errorf("Parameter QN missing inner class scope: %s", defs[0].Element.QualifiedName)
		}

		// 验证 Signature 格式: long duration
		if !strings.Contains(defs[0].Element.Signature, "long duration") {
			t.Errorf("Invalid signature: %s", defs[0].Element.Signature)
		}
	})
}

func TestJavaCollector_ScopeAndShadowing(t *testing.T) {
	// 1. 初始化解析环境
	filePath := getTestFilePath(filepath.Join("com", "example", "base", "test", "ScopeTest.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	printCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证方法直接作用域下的变量 x (第一次出现，无后缀)
	t.Run("Verify Root Method Variable", func(t *testing.T) {
		qn := "com.example.base.ScopeTest.test().x" // 修正：移除 $1
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Root variable x not found")
		}
	})

	// 2. 验证第一个独立代码块 { int x = 2; }
	t.Run("Verify First Block Shadowing", func(t *testing.T) {
		// block 始终带 $n，但 block 内部的第一个 x 不带 $n
		qn := "com.example.base.ScopeTest.test().block$1.x"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable in block$1 not found")
		}
	})

	// 3. 验证 if 分支代码块 { int x = 3; }
	t.Run("Verify If-Statement Block", func(t *testing.T) {
		qn := "com.example.base.ScopeTest.test().block$2.x"
		defs := findDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable in block$2 not found")
		}
	})

	// 4. 验证 Lambda 表达式及其内部变量
	t.Run("Verify Lambda Scope", func(t *testing.T) {
		// 根据日志，Lambda 内部还有一个自动生成的 block$1
		// QN: com.example.base.ScopeTest.test().lambda$1.block$1.x
		lambdaVarQN := "com.example.base.ScopeTest.test().lambda$1.block$1.x"
		if len(findDefinitionsByQN(fCtx, lambdaVarQN)) == 0 {
			t.Errorf("Variable x not found inside Lambda block scope")
		}
	})
}

// 辅助函数：根据 QN 在 fCtx 中查找定义
func findDefinitionsByQN(fCtx *core.FileContext, targetQN string) []*core.DefinitionEntry {
	var result []*core.DefinitionEntry
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.Element.QualifiedName == targetQN {
				result = append(result, entry)
			}
		}
	}

	return result
}

// 辅助函数：判断 slice 是否包含 string
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
