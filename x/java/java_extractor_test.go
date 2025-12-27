package java_test

import (
	"fmt"
	"path/filepath"
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

const printRel = false

func printRelations(relations []*model.DependencyRelation) {
	if !printRel {
		return
	}

	for _, rel := range relations {
		fmt.Printf("Found relation: [%s] -> source(%s)==>target(%s)\n", rel.Type, rel.Source.QualifiedName, rel.Target.QualifiedName)
	}
}

// 验证结构关系：CONTAIN
// 验证行为关系：CREATE（由匿名内部类产生）、Call（由匿名内部类产生）
func TestJavaExtractor_CallbackManager(t *testing.T) {
	// 1. 准备测试文件路径
	targetFile := getTestFilePath(filepath.Join("com", "example", "service", "CallbackManager.java"))

	// 2. Phase 1: 构建全局上下文 (运行已完成的 Collector)
	gCtx := runPhase1Collection(t, []string{targetFile})

	// 3. Phase 2: 运行 Extractor
	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(relations)

	// 4. 验证依赖关系
	t.Run("Verify Structural Relations", func(t *testing.T) {
		foundLocalClass := false
		foundMethodInLocal := false
		foundRegisterMethod := false

		for _, rel := range relations {
			// 期望 1: CallbackManager.register -> CONTAIN -> LocalValidator
			if rel.Type == model.Contain &&
				rel.Source.Name == "register" &&
				rel.Target.Name == "LocalValidator" {
				foundLocalClass = true
				assert.Equal(t, model.Class, rel.Target.Kind)
				assert.Equal(t, "com.example.service.CallbackManager.register.LocalValidator", rel.Target.QualifiedName)
			}

			// 期望 2: LocalValidator -> CONTAIN -> isValid
			if rel.Type == model.Contain &&
				rel.Source.Name == "LocalValidator" &&
				rel.Target.Name == "isValid" {
				foundMethodInLocal = true
				assert.Equal(t, model.Method, rel.Target.Kind)
			}

			// 期望 3: CallbackManager -> CONTAIN -> register
			if rel.Type == model.Contain &&
				rel.Source.Name == "CallbackManager" &&
				rel.Target.Name == "register" {
				foundRegisterMethod = true
			}
		}

		assert.True(t, foundRegisterMethod, "Should find CallbackManager -> register")
		assert.True(t, foundLocalClass, "Should find register -> LocalValidator (Local Class)")
		assert.True(t, foundMethodInLocal, "Should find LocalValidator -> isValid")
	})

	t.Run("Verify Robustness (Anonymous Class)", func(t *testing.T) {
		// 匿名内部类 Runnable 应该产生 CREATE 关系，但不应该在 CONTAIN 关系中出现（因为它没有名字）
		foundCreateRunnable := false
		for _, rel := range relations {
			// 匿名类创建：new Runnable() { ... }
			// 由于 Runnable 是外部接口，这里 Target 的 QN 可能就是 "Runnable"
			if rel.Type == model.Create && rel.Target.Name == "Runnable" {
				foundCreateRunnable = true
			}

			// 验证不存在空名称的 Contain 关系
			if rel.Type == model.Contain {
				assert.NotEmpty(t, rel.Target.Name, "Target name in CONTAIN relation should not be empty")
			}
		}
		assert.True(t, foundCreateRunnable, "Should capture creation of anonymous Runnable")
	})

	t.Run("Verify Action Relations", func(t *testing.T) {
		foundPrintCall := false
		for _, rel := range relations {
			// 验证匿名类内部的方法调用：System.out.println
			if rel.Type == model.Call && rel.Target.Name == "println" {
				foundPrintCall = true

				// Source 应该是匿名内部类里的 run 方法
				assert.Equal(t, "run", rel.Source.Name, "The direct source of println should be the run method")

				// 进一步验证 QN，确保它嵌套在 register 下
				// 预期路径: 包名.CallbackManager.register.run (取决于你 Collector 的递归逻辑)
				assert.Contains(t, rel.Source.QualifiedName, "CallbackManager.register",
					"The run method should be scoped under register")
			}
		}
		assert.True(t, foundPrintCall, "Should capture println call inside anonymous class")
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

	// 2. Phase 1: 构建全局上下文 (Symbol Table)
	gCtx := runPhase1Collection(t, testFiles)
	targetFile := testFiles[0]

	// 3. Phase 2: 运行 Extractor
	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(relations)

	// 4. 验证核心结构化关系 (已解决泛型和注解匹配问题)
	t.Run("Verify Structural Relations (Fixed QN)", func(t *testing.T) {
		expectations := []struct {
			relType model.DependencyType
			target  string // 期望全限定名
			kind    model.ElementKind
		}{
			// 测试 EXTEND 清洗: AbstractBaseEntity<String> -> com.example.model.AbstractBaseEntity
			{model.Extend, "com.example.model.AbstractBaseEntity", model.Class},
			// 测试 IMPLEMENT 清洗: DataProcessor<...> -> com.example.core.DataProcessor
			{model.Implement, "com.example.core.DataProcessor", model.Interface},
			// 测试 ANNOTATION 清洗: @Loggable -> com.example.annotation.Loggable
			{model.Annotation, "com.example.annotation.Loggable", model.KAnnotation},
		}

		for _, exp := range expectations {
			found := false
			for _, rel := range relations {
				if rel.Type == exp.relType && rel.Target.QualifiedName == exp.target {
					found = true
					assert.Equal(t, exp.kind, rel.Target.Kind)
					break
				}
			}
			assert.True(t, found, "Missing Structural Relation: [%s] -> %s", exp.relType, exp.target)
		}
	})

	// 5. 验证新增的丰满关系 (PARAMETER, RETURN, THROW)
	t.Run("Verify Rich Method Relations", func(t *testing.T) {
		hasReturn := false
		hasParam := false
		hasThrow := false
		hasOverride := false

		for _, rel := range relations {
			// 验证 processAll 方法的返回类型 (List)
			if rel.Type == model.Return && rel.Target.Name == "List" && rel.Source.Name == "processAll" {
				hasReturn = true
			}
			// 验证 processAll 方法的参数类型 (String)
			if rel.Type == model.Parameter && rel.Target.Name == "String" && rel.Source.Name == "processAll" {
				hasParam = true
			}
			// 验证 processAll 抛出的异常 (RuntimeException)
			if rel.Type == model.Throw && rel.Target.Name == "RuntimeException" {
				hasThrow = true
			}
			// 验证方法上的注解 (@Override)
			if rel.Type == model.Annotation && rel.Target.Name == "Override" && rel.Source.Name == "processAll" {
				hasOverride = true
			}
		}

		assert.True(t, hasReturn, "Should extract RETURN relation for processAll")
		assert.True(t, hasParam, "Should extract PARAMETER relation for batchId")
		assert.True(t, hasThrow, "Should extract THROW relation for RuntimeException")
		assert.True(t, hasOverride, "Should extract ANNOTATION relation for @Override")
	})

	// 6. 验证行为关系 (Actions)
	t.Run("Verify Action Relations", func(t *testing.T) {
		foundCreate := false
		foundCall := false
		foundCast := false

		for _, rel := range relations {
			// new ArrayList<>()
			if rel.Type == model.Create && rel.Target.Name == "ArrayList" {
				foundCreate = true
			}
			// batchId.toUpperCase()
			if rel.Type == model.Call && rel.Target.Name == "toUpperCase" {
				foundCall = true
				assert.Equal(t, "processAll", rel.Source.Name)
			}
			// (String) rawData
			if rel.Type == model.Cast && rel.Target.Name == "String" {
				foundCast = true
			}
		}
		assert.True(t, foundCreate, "Should capture Create relation")
		assert.True(t, foundCall, "Should capture Call relation")
		assert.True(t, foundCast, "Should capture Cast relation")
	})

	// 7. 验证字段访问与 Source 溯源
	t.Run("Verify Field Access and Constructor Source", func(t *testing.T) {
		foundFieldUse := false
		for _, rel := range relations {
			// 构造函数内的 this.id 访问
			if rel.Type == model.Use && rel.Target.Name == "id" {
				foundFieldUse = true
				// 构造函数的 Source Name 应该被识别为 UserServiceImpl
				assert.Equal(t, "UserServiceImpl", rel.Source.Name)
			}
		}
		assert.True(t, foundFieldUse, "Should capture Use relation for 'id' in constructor")
	})
}
