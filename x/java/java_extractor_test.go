package java_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java" // 触发 init() 注册
	"github.com/stretchr/testify/assert"
)

// 辅助函数：解析并收集定义 (Phase 1)
func runPhase1Collection(t *testing.T, files []string) *model.GlobalContext {
	gc := model.NewGlobalContext()
	javaParser, err := parser.NewParser(model.LangJava)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer javaParser.Close()

	col := java.NewJavaCollector()

	for _, file := range files {
		rootNode, sourceBytes, err := javaParser.ParseFile(file, false, false)
		if err != nil {
			t.Fatalf("Failed to parse file %s: %v", file, err)
		}

		fCtx, err := col.CollectDefinitions(rootNode, file, sourceBytes)
		if err != nil {
			t.Fatalf("Failed to collect definitions for %s: %v", file, err)
		}
		gc.RegisterFileContext(fCtx)
	}
	return gc
}

const printRel = true

func printRelations(relations []*model.DependencyRelation) {
	if !printRel {
		return
	}

	for _, rel := range relations {
		fmt.Printf("Found relation: [%s] -> source(%s::%s)==>target(%s::%s)\n", rel.Type, rel.Source.Kind, rel.Source.QualifiedName, rel.Target.Kind, rel.Target.QualifiedName)
	}
}

// 验证结构关系：CONTAIN
// 验证行为关系：CREATE（由匿名内部类产生）、Call（由匿名内部类产生）
// 验证 JDK 内置符号解析
func TestJavaExtractor_CallbackManager(t *testing.T) {
	// 1. 准备测试文件路径
	targetFile := getTestFilePath(filepath.Join("com", "example", "service", "CallbackManager.java"))

	// 2. Phase 1: 构建全局上下文 (运行 Collector)
	// 即使只有一个文件，也需要运行 Phase 1 以便 Extractor 获取 FileContext
	gCtx := runPhase1Collection(t, []string{targetFile})

	// 3. Phase 2: 运行增强后的 Extractor
	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}
	printRelations(relations)

	// 4. 验证顶层 Package 与 File 关系 (新增功能)
	t.Run("Verify Package and File Relations", func(t *testing.T) {
		foundPkgFile := false
		for _, rel := range relations {
			if rel.Type == model.Contain &&
				rel.Source.Kind == model.Package &&
				rel.Source.Name == "com.example.service" &&
				rel.Target.Kind == model.File {
				foundPkgFile = true
				break
			}
		}
		assert.True(t, foundPkgFile, "Should find Package -> CONTAIN -> File relation")
	})

	// 5. 验证局部类 (Local Class) 的层级提取
	t.Run("Verify Local Class Structure", func(t *testing.T) {
		foundLocalClass := false
		for _, rel := range relations {
			// register 方法应该 CONTAIN LocalValidator
			if rel.Type == model.Contain &&
				rel.Source.Name == "register" &&
				rel.Target.Name == "LocalValidator" {
				foundLocalClass = true
				assert.Equal(t, model.Class, rel.Target.Kind)
				// 验证 QN 是否正确包含了方法名作为前缀
				assert.Contains(t, rel.Target.QualifiedName, "CallbackManager.register.LocalValidator")
			}
		}
		assert.True(t, foundLocalClass, "Should extract LocalValidator under register method")
	})

	// 6. 验证 JDK 内置符号解析 (JavaBuiltinTable)
	t.Run("Verify JDK Builtin Resolution", func(t *testing.T) {
		foundSystemOut := false
		foundRunnableType := false

		for _, rel := range relations {
			// 验证 System.out 的解析
			if rel.Type == model.Use && rel.Target.Name == "out" {
				foundSystemOut = true
				assert.Equal(t, "java.lang.System.out", rel.Target.QualifiedName)
				assert.Equal(t, model.Field, rel.Target.Kind)
			}

			// 验证 Runnable 接口类型的解析 (通过 JavaBuiltinTable)
			if rel.Target.Name == "Runnable" {
				foundRunnableType = true
				assert.Equal(t, "java.lang.Runnable", rel.Target.QualifiedName)
				assert.Equal(t, model.Interface, rel.Target.Kind)
			}
		}
		assert.True(t, foundSystemOut, "Should resolve 'out' to java.lang.System.out")
		assert.True(t, foundRunnableType, "Should resolve 'Runnable' to java.lang.Runnable")
	})

	// 7. 验证匿名内部类中的方法调用归属
	t.Run("Verify Chained Call Resolution", func(t *testing.T) {
		foundFullCall := false
		for _, rel := range relations {
			if rel.Type == model.Call && rel.Target.QualifiedName == "java.lang.System.out.println" {
				foundFullCall = true
				break
			}
		}
		assert.True(t, foundFullCall, "Should resolve full path for chained call: java.lang.System.out.println")
	})
}

// 验证结构关系：Extend（泛型）、Implement（泛型）、ANNOTATION
// 验证结构关系：Return、Parameter、Throw
// 验证行为关系：Create、Call、Cast
// 验证字段访问： Use
func TestJavaExtractor_UserServiceImpl(t *testing.T) {
	// 1. 准备环境：加载目标文件及其依赖项以确保 QN 解析成功
	testFiles := []string{
		getTestFilePath(filepath.Join("com", "example", "service", "UserServiceImpl.java")),
		getTestFilePath(filepath.Join("com", "example", "model", "AbstractBaseEntity.java")),
		getTestFilePath(filepath.Join("com", "example", "core", "DataProcessor.java")),
		getTestFilePath(filepath.Join("com", "example", "annotation", "Loggable.java")),
	}

	// 2. Phase 1: 构建符号表
	gCtx := runPhase1Collection(t, testFiles)
	targetFile := testFiles[0]

	// 3. Phase 2: 运行 Extractor
	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(relations)

	// 4. 验证顶层文件关系
	t.Run("Verify File and Package Relations", func(t *testing.T) {
		hasPackageContain := false
		hasImport := false

		for _, rel := range relations {
			// 验证 Package -> CONTAIN -> File
			if rel.Type == model.Contain && rel.Source.Kind == model.Package {
				if rel.Source.QualifiedName == "com.example.service" && rel.Target.Kind == model.File {
					hasPackageContain = true
				}
			}

			// 验证 File -> IMPORT -> AbstractBaseEntity (全路径验证)
			if rel.Type == model.Import && rel.Source.Kind == model.File {
				if rel.Target.QualifiedName == "com.example.model.AbstractBaseEntity" {
					hasImport = true
				}
			}
		}

		assert.True(t, hasPackageContain, "Should find Package -> CONTAIN -> File")
		assert.True(t, hasImport, "Should find File -> IMPORT -> com.example.model.AbstractBaseEntity")
	})

	// 5. 验证结构化关系 (PARAMETER, RETURN, THROW, ANNOTATION)
	t.Run("Verify Rich Method Relations", func(t *testing.T) {
		hasReturn := false
		hasParam := false
		hasThrow := false
		hasOverride := false

		for _, rel := range relations {
			// 验证 processAll 方法的返回类型 (List -> java.util.List via Import)
			if rel.Type == model.Return && rel.Source.Name == "processAll" {
				if rel.Target.QualifiedName == "java.util.List" {
					hasReturn = true
				}
			}
			// 验证 processAll 方法的参数类型 (String -> java.lang.String via Heuristic)
			if rel.Type == model.Parameter && rel.Source.Name == "processAll" {
				if rel.Target.QualifiedName == "java.lang.String" {
					hasParam = true
				}
			}
			// 验证 THROW 关系 (RuntimeException -> java.lang.RuntimeException)
			if rel.Type == model.Throw && rel.Source.Name == "processAll" {
				if rel.Target.QualifiedName == "java.lang.RuntimeException" {
					hasThrow = true
				}
			}
			// 验证 Override 注解 (Override -> java.lang.Override)
			if rel.Type == model.Annotation && rel.Source.Name == "processAll" {
				if rel.Target.QualifiedName == "java.lang.Override" {
					hasOverride = true
				}
			}
		}

		assert.True(t, hasReturn, "Should extract RETURN java.util.List for processAll")
		assert.True(t, hasParam, "Should extract PARAMETER java.lang.String for batchId")
		assert.True(t, hasThrow, "Should extract THROW java.lang.RuntimeException")
		assert.True(t, hasOverride, "Should extract ANNOTATION java.lang.Override")
	})

	// 6. 验证行为关系 (Actions: CALL, CREATE, CAST)
	t.Run("Verify Action Relations", func(t *testing.T) {
		foundCreate := false
		foundCall := false
		foundUUIDCall := false
		foundCast := false

		for _, rel := range relations {
			// new ArrayList<>()
			if rel.Type == model.Create && rel.Target.Name == "ArrayList" {
				foundCreate = true
			}
			// batchId.toUpperCase() -> QN 应包含 String 的路径前缀或本身
			if rel.Type == model.Call && rel.Target.QualifiedName == "batchId.toUpperCase" {
				foundCall = true
				assert.Equal(t, "processAll", rel.Source.Name)
			}
			// UUID.randomUUID() -> 验证 BuiltinTable 映射
			if rel.Type == model.Call && rel.Target.QualifiedName == "java.util.UUID.randomUUID" {
				foundUUIDCall = true
			}
			// (String) rawData
			if rel.Type == model.Cast && rel.Target.QualifiedName == "java.lang.String" {
				foundCast = true
			}
		}
		assert.True(t, foundCreate, "Should capture Create relation")
		assert.True(t, foundCall, "Should capture Call to toUpperCase")
		assert.True(t, foundUUIDCall, "Should capture Call to java.util.UUID.randomUUID")
		assert.True(t, foundCast, "Should capture Cast to java.lang.String")
	})

	// 7. 验证字段访问与构造函数内的 this 处理
	t.Run("Verify Field Access and Constructor Source", func(t *testing.T) {
		foundFieldUse := false
		for _, rel := range relations {
			// 构造函数 UserServiceImpl 内部对 id 的访问
			if rel.Type == model.Use && rel.Target.Name == "id" {
				if rel.Source.Name == "UserServiceImpl" {
					foundFieldUse = true
				}
			}
		}
		assert.True(t, foundFieldUse, "Should capture Use relation for 'id' in constructor with correct Source")
	})
}

func TestJavaExtractor_ModernFeatures(t *testing.T) {
	targetFile := getTestFilePath(filepath.Join("com", "example", "shop", "ModernOrderProcessor.java"))
	gCtx := runPhase1Collection(t, []string{targetFile})

	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	assert.NoError(t, err)

	t.Run("Verify Record Definitions", func(t *testing.T) {
		// 验证是否同时生成了 Field 和 Method (Accessor)
		foundField := false
		foundAccessor := false
		for _, defs := range gCtx.FileContexts[targetFile].DefinitionsBySN {
			for _, d := range defs {
				if d.Element.Name == "price" {
					if d.Element.Kind == model.Field {
						foundField = true
					}
					if d.Element.Kind == model.Method {
						foundAccessor = true
					}
				}
			}
		}
		assert.True(t, foundField, "Record component 'price' should be a Field")
		assert.True(t, foundAccessor, "Record component 'price' should also be a Method")
	})

	t.Run("Verify Method Reference Resolution", func(t *testing.T) {
		foundRef := false
		for _, rel := range relations {
			// 验证 Order::price 是否解析为 com.example.shop.Order.price
			if rel.Type == model.Call && rel.Target.QualifiedName == "com.example.shop.Order.price" {
				foundRef = true
			}
		}
		assert.True(t, foundRef, "Should resolve method reference Order::price to full QN")
	})

	t.Run("Verify System Out Method Reference", func(t *testing.T) {
		foundSystemRef := false
		for _, rel := range relations {
			// 验证 System.out::println
			if rel.Type == model.Call && rel.Target.QualifiedName == "java.lang.System.out.println" {
				foundSystemRef = true
			}
		}
		assert.True(t, foundSystemRef, "Should resolve System.out::println using builtin table and reference logic")
	})

	t.Run("Verify Chained Accessor Call", func(t *testing.T) {
		foundIdCall := false
		for _, rel := range relations {
			// 验证 this.id() 调用
			if rel.Type == model.Call && strings.Contains(rel.Target.QualifiedName, "Order.id") {
				foundIdCall = true
			}
		}
		assert.True(t, foundIdCall, "Should capture call to implicit record accessor id()")
	})
}
