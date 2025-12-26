package java_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	// 导入所有语言绑定，确保 GetLanguage 可以找到
	_ "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func getTestFilePath(name string) string {
	currentDir, _ := filepath.Abs(filepath.Dir("."))
	return filepath.Join(currentDir, "testdata", name)
}

func getJavaParser(t *testing.T) *parser.TreeSitterParser {
	javaParser, err := parser.NewParser(model.LangJava)
	if err != nil {
		t.Fatalf("Failed to create Java parser: %v", err)
	}

	return javaParser
}

// 验证注解定义、元注解提取
// 验证语义化 Import
// 验证注释提取
func TestJavaCollector_LoggableAnnotation(t *testing.T) {
	// 1. 获取测试文件路径 (对应 x/java/testdata/com/example/annotation/Loggable.java)
	filePath := getTestFilePath(filepath.Join("com", "example", "annotation", "Loggable.java"))

	// 2. 解析源码
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, true, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 3. 运行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	// 4. 验证 Package Name
	expectedPackage := "com.example.annotation"
	if fCtx.PackageName != expectedPackage {
		t.Errorf("Expected PackageName %q, got %q", expectedPackage, fCtx.PackageName)
	}

	// 5. 验证语义化 Import (修正重点：从 map[string]string 变为 ImportEntry)
	t.Run("Verify Semantic Imports", func(t *testing.T) {
		imp, ok := fCtx.Imports["*"]
		if !ok {
			t.Fatal("Expected wildcard import '*' not found")
		}
		if imp.RawImportPath != "java.lang.annotation.*" {
			t.Errorf("Import path mismatch: expected java.lang.annotation.*, got %s", imp.RawImportPath)
		}
		if !imp.IsWildcard {
			t.Error("Expected IsWildcard to be true")
		}
		if imp.Kind != model.Package {
			t.Errorf("Expected Kind %s, got %s", model.Package, imp.Kind)
		}
	})

	// 6. 验证注解定义与注释 (修正重点：增加 Doc 验证)
	t.Run("Verify Annotation and Doc", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["Loggable"]
		if len(defs) == 0 {
			t.Fatal("Annotation 'Loggable' not found")
		}

		elem := defs[0].Element
		if elem.Kind != model.KAnnotation {
			t.Errorf("Expected Kind %s, got %s", model.KAnnotation, elem.Kind)
		}

		// 验证 Doc 注释提取
		if !strings.Contains(elem.Doc, "测试：Annotation Type Declaration") {
			t.Errorf("Doc mismatch, got: %q", elem.Doc)
		}

		// 验证元注解提取
		if elem.Extra == nil || len(elem.Extra.Annotations) == 0 {
			t.Error("Extra.Annotations should not be empty")
		}
	})

	// 7. 验证注解属性 (修正重点：现在应该能通过了)
	t.Run("Verify Annotation Properties", func(t *testing.T) {
		properties := []struct {
			name string
			ret  string
			sign string
		}{
			{"level", "String", "String level()"},
			{"trace", "boolean", "boolean trace()"},
		}

		for _, prop := range properties {
			defs := fCtx.DefinitionsBySN[prop.name]
			if len(defs) == 0 {
				t.Errorf("Annotation property %q not found. Check if node type 'annotation_type_element_declaration' is handled.", prop.name)
				continue
			}

			elem := defs[0].Element
			if elem.Kind != model.Method {
				t.Errorf("Property %q: expected Kind METHOD, got %s", prop.name, elem.Kind)
			}

			// 验证返回类型 (存在于 Extra 和 Signature 中)
			if elem.Extra == nil || elem.Extra.MethodExtra.ReturnType != prop.ret {
				t.Errorf("Property %q: expected return type %q, got %v", prop.name, prop.ret, elem.Extra.MethodExtra.ReturnType)
			}

			// 验证细化后的 Signature
			if !strings.Contains(elem.Signature, prop.sign) {
				t.Errorf("Property %q: signature mismatch. Expected contains %q, got %q", prop.name, prop.sign, elem.Signature)
			}
		}
	})

	fmt.Println("Java Collector Loggable test completed successfully.")
}

// 验证包名与导入
// 验证顶级类定义
// 验证字段与方法
// 验证嵌套内部类 (最关键的 QN 逻辑)
func TestJavaCollector_AbstractBaseEntity(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := getTestFilePath(filepath.Join("com", "example", "model", "AbstractBaseEntity.java"))

	// 2. 解析源码
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, true, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 3. 运行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	// 4. 验证包名与导入
	assert.Equal(t, "com.example.model", fCtx.PackageName)
	assert.Contains(t, fCtx.Imports, "Serializable")
	assert.Contains(t, fCtx.Imports, "Date")

	// 5. 验证顶级类定义
	t.Run("Verify Top Level Class", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["AbstractBaseEntity"]
		require.NotEmpty(t, defs)

		elem := defs[0].Element
		assert.Equal(t, model.Class, elem.Kind)
		assert.Equal(t, "com.example.model.AbstractBaseEntity", elem.QualifiedName)

		// 验证 ClassExtra (继承与实现)
		require.NotNil(t, elem.Extra.ClassExtra)
		// 注意：Java AST 可能会保留泛型符号 <ID>，根据你目前的 getNodeContent 逻辑判断
		assert.Contains(t, elem.Extra.ClassExtra.ImplementedInterfaces, "Serializable")
		assert.Contains(t, elem.Extra.Modifiers, "public")
		assert.Contains(t, elem.Extra.Modifiers, "abstract")
	})

	// 6. 验证字段与方法
	t.Run("Verify Members", func(t *testing.T) {
		// 验证字段 id
		idDefs := fCtx.DefinitionsBySN["id"]
		require.NotEmpty(t, idDefs)
		assert.Equal(t, "com.example.model.AbstractBaseEntity.id", idDefs[0].Element.QualifiedName)
		assert.Equal(t, "ID", idDefs[0].Element.Extra.FieldExtra.Type)

		// 验证方法 getId
		getDefs := fCtx.DefinitionsBySN["getId"]
		require.NotEmpty(t, getDefs)
		assert.Equal(t, "com.example.model.AbstractBaseEntity.getId", getDefs[0].Element.QualifiedName)
		assert.Equal(t, "ID", getDefs[0].Element.Extra.MethodExtra.ReturnType)
	})

	// 7. 验证嵌套内部类 (最关键的 QN 逻辑)
	t.Run("Verify Nested Inner Class", func(t *testing.T) {
		// 验证内部类 EntityMeta
		metaDefs := fCtx.DefinitionsBySN["EntityMeta"]
		require.NotEmpty(t, metaDefs)

		metaElem := metaDefs[0].Element
		assert.Equal(t, model.Class, metaElem.Kind)
		// 验证 QN 是否正确拼接了父类名
		expectedMetaQN := "com.example.model.AbstractBaseEntity.EntityMeta"
		assert.Equal(t, expectedMetaQN, metaElem.QualifiedName)

		// 验证内部类的成员
		tableDefs := fCtx.DefinitionsBySN["tableName"]
		require.NotEmpty(t, tableDefs)
		assert.Equal(t, expectedMetaQN+".tableName", tableDefs[0].Element.QualifiedName)
		assert.Equal(t, "String", tableDefs[0].Element.Extra.FieldExtra.Type)
	})

	// 8. 验证 Doc 采集
	t.Run("Verify Class Doc", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["AbstractBaseEntity"]
		if len(defs) > 0 {
			assert.Contains(t, defs[0].Element.Doc, "基础实体类")
		}
	})
}

// 验证接口继承
// 验证方法异常抛出
// 验证默认方法与修饰符
func TestJavaCollector_DataProcessor_Complex(t *testing.T) {
	// 1. 初始化
	filePath := getTestFilePath(filepath.Join("com", "example", "core", "DataProcessor.java"))
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, true, false)
	require.NoError(t, err)

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	require.NoError(t, err)

	// 2. 验证接口继承 (extends Runnable, AutoCloseable)
	t.Run("Verify Multiple Interface Inheritance", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["DataProcessor"]
		require.NotEmpty(t, defs)

		ce := defs[0].Element.Extra.ClassExtra
		require.NotNil(t, ce)

		// 验证继承的接口列表
		// 在 Java 中，接口继承其他接口使用的是 extends 关键字，但在 AST 中对应 interfaces 字段
		assert.Contains(t, ce.ImplementedInterfaces, "Runnable")
		assert.Contains(t, ce.ImplementedInterfaces, "AutoCloseable")

		// 验证 Doc 采集是否包含了测试描述
		assert.Contains(t, defs[0].Element.Doc, "Method Throws")
	})

	// 3. 验证方法异常抛出 (throws RuntimeException, Exception)
	t.Run("Verify Method Throws and Return Type", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["processAll"]
		require.NotEmpty(t, defs)

		elem := defs[0].Element
		me := elem.Extra.MethodExtra
		require.NotNil(t, me)

		// 验证返回类型 (带泛型的引用)
		assert.Equal(t, "List<T>", me.ReturnType)

		// 验证 Throws 异常列表
		assert.Equal(t, 2, len(me.ThrowsTypes))
		assert.Contains(t, me.ThrowsTypes, "RuntimeException")
		assert.Contains(t, me.ThrowsTypes, "Exception")
	})

	// 4. 验证默认方法与修饰符
	t.Run("Verify Default Method", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["stop"]
		require.NotEmpty(t, defs)

		elem := defs[0].Element
		// 验证是否识别出 default 关键字
		assert.Contains(t, elem.Extra.Modifiers, "default")

		// 验证签名完整性
		assert.Equal(t, "default void stop()", elem.Signature)
	})

	// 5. 验证导入表
	t.Run("Verify Imports", func(t *testing.T) {
		assert.Contains(t, fCtx.Imports, "AbstractBaseEntity")
		assert.Equal(t, "com.example.model.AbstractBaseEntity", fCtx.Imports["AbstractBaseEntity"].RawImportPath)

		assert.Contains(t, fCtx.Imports, "List")
		assert.Equal(t, "java.util.List", fCtx.Imports["List"].RawImportPath)
	})
}
