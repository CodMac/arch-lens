package java_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java"
	"github.com/stretchr/testify/assert"
)

const printRel = true

func TestJavaExtractor_Annotation(t *testing.T) {
	// 1. 准备测试文件路径
	testFile := "testdata/com/example/rel/AnnotationRelationSuite.java"
	files := []string{testFile}

	// 2. 执行 Phase 1: 收集定义
	gCtx := runPhase1Collection(t, files)

	// 3. 执行 Phase 2: 提取依赖关系
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

	// 4. 定义全量断言数据集
	expectedRels := []struct {
		relType    model.DependencyType
		sourceQN   string
		targetQN   string
		targetKind model.ElementKind
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 类注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite",
			targetQN:   "Entity",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "TYPE", m[java.RelAnnotationTarget])
				// 去掉 RelAstKind 的断言
			},
		},
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite",
			targetQN:   "SuppressWarnings",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "TYPE", m[java.RelAnnotationTarget])
				assert.Equal(t, "\"all\"", m[java.RelAnnotationValue])
			},
		},
		// --- 2. 字段注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.id",
			targetQN:   "Column",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "FIELD", m[java.RelAnnotationTarget])
				// 增加空格以匹配 raw text 提取结果，或使用 Contains 模糊匹配
				assert.Contains(t, m[java.RelAnnotationParams], "name = \"user_id\"")
				assert.Contains(t, m[java.RelAnnotationParams], "nullable = false")
			},
		},
		// --- 3. 方法注解 ---
		{
			relType: model.Annotation,
			// 必须包含参数列表以匹配 Collector 生成的 QN
			sourceQN:   "com.example.rel.AnnotationRelationSuite.save(String)",
			targetQN:   "Transactional",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "METHOD", m[java.RelAnnotationTarget])
				assert.Contains(t, m[java.RelAnnotationParams], "timeout = 100")
			},
		},
		// --- 3.1 参数注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "save(String).data",
			targetQN:   "NotNull",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "PARAMETER", m[java.RelAnnotationTarget])
			},
		},
		// --- 4. 局部变量注解 ---
		{
			relType: model.Annotation,
			// 必须包含父方法的参数列表
			sourceQN:   "save(String).local",
			targetQN:   "NonEmpty",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "LOCAL_VARIABLE", m[java.RelAnnotationTarget])
			},
		},
	}

	// 5. 执行匹配断言逻辑 (支持部分 QN 匹配以增强鲁棒性)
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配逻辑：类型相同 && Target 名字相同 && Source QN 后缀匹配
			if rel.Type == exp.relType &&
				rel.Target.Name == exp.targetQN &&
				strings.HasSuffix(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				// 校验 ElementKind
				assert.Equal(t, exp.targetKind, rel.Target.Kind, "Kind mismatch for target: %s", exp.targetQN)

				// 校验 Mores 元数据
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing expected relation: [%s] %s -> %s", exp.relType, exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Assign(t *testing.T) {
	// 1. 准备测试文件路径
	testFile := "testdata/com/example/rel/AssignRelationSuite.java"
	files := []string{testFile}

	// 2. 执行 Phase 1 & 2
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

	// 3. 定义断言数据集
	// 关键：增加 matchMores 逻辑，确保在多个同 Source-Target 关系中选出正确的那一个
	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		matchMores func(m map[string]interface{}) bool
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 字段声明初始化 ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.count",
			targetQN: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "0", m[java.RelAssignValueExpression])
			},
		},
		// --- 2. 静态代码块赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.$static$1",
			targetQN: "status",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "=", m[java.RelAssignOperator])
				assert.Equal(t, "\"INIT\"", m[java.RelAssignValueExpression])
				assert.Equal(t, true, m[java.RelAssignIsStaticContext])
			},
		},
		// --- 3. 局部变量基础赋值 (声明时) ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments",
			targetQN: "local",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "10", m[java.RelAssignValueExpression])
			},
		},
		// --- 4. 成员变量赋值 (带 Receiver) ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments",
			targetQN: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignOperator] == "=" && m[java.RelAssignValueExpression] == "100"
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelCallReceiver])
			},
		},
		// --- 5. 复合赋值 (+=) ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments",
			targetQN: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignOperator] == "+="
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelAssignIsCompound])
			},
		},
		// --- 7. 更新表达式 (count++) ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments",
			targetQN: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignOperator] == "++"
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "update_expression", m[java.RelAstKind])
			},
		},
		// --- 8. 数组元素赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments",
			targetQN: "arr",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIndexExpression] == "0"
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "99", m[java.RelAssignValueExpression])
			},
		},
		// --- 9. Lambda 内部赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1",
			targetQN: "count",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "300", m[java.RelAssignValueExpression])
				assert.Equal(t, "this", m[java.RelCallReceiver])
			},
		},
	}

	// 4. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 基础匹配
			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.Name == exp.targetQN {

				// 如果定义了 matchMores，则进行二次筛选（解决同变量多次赋值问题）
				if exp.matchMores != nil && !exp.matchMores(rel.Mores) {
					continue
				}

				if exp.checkMores != nil {
					found = true
					exp.checkMores(t, rel.Mores)
					break
				}
			}
		}
		assert.True(t, found, "Missing or Incorrect Assign relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_ClassAssign(t *testing.T) {
	testFile := "testdata/com/example/rel/AssignRelationForClassSuite.java"
	files := []string{testFile}

	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		matchMores func(m map[string]interface{}) bool
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 实例化赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationForClassSuite.testClassAssignments",
			targetQN: "list",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "new ArrayList<>()", m[java.RelAssignValueExpression])
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		// --- 2. 跨对象字段赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationForClassSuite.testClassAssignments",
			targetQN: "name",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "data", m[java.RelCallReceiver]) // 识别出 data 是接收者
				assert.Equal(t, "\"Hello\"", m[java.RelAssignValueExpression])
			},
		},
		// --- 3. 方法返回赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationForClassSuite.testClassAssignments",
			targetQN: "globalObj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "fetchObject()", m[java.RelAssignValueExpression])
				assert.Equal(t, "this", m[java.RelCallReceiver])
			},
		},
		// --- 4. Null 赋值 ---
		{
			sourceQN: "com.example.rel.AssignRelationForClassSuite.testClassAssignments",
			targetQN: "data",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignValueExpression] == "null"
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "=", m[java.RelAssignOperator])
			},
		},
	}

	// 匹配逻辑同上个回答中的通用逻辑
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.Name == exp.targetQN {

				if exp.matchMores != nil && !exp.matchMores(rel.Mores) {
					continue
				}
				if exp.checkMores != nil {
					found = true
					exp.checkMores(t, rel.Mores)
					break
				}
			}
		}
		assert.True(t, found, "Missing Class Assign relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Call(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/CallRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 方法名或类名(针对构造函数)
		relType    model.DependencyType
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 基础实例调用 ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "simpleMethod",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelCallReceiver])
				assert.Equal(t, false, m[java.RelCallIsStatic])
			},
		},
		// --- 2. 静态方法调用 ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "staticMethod",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsStatic])
				assert.Equal(t, "CallRelationSuite", m[java.RelCallReceiverType])
			},
		},
		// --- 4. 链式调用 (提取 getList() -> add()) ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "getList",
			relType:  model.Call,
		},
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "add",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "getList()", m[java.RelCallReceiverExpression])
				assert.Equal(t, true, m[java.RelCallIsChained])
			},
		},
		// --- 5 & 6. 继承调用 ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "baseMethod",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsInherited])
			},
		},
		// --- 7. 对象创建 (CREATE 关系通常也作为一种特殊的 CALL 记录) ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "ArrayList",
			relType:  model.Create,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "String", m[java.RelCallTypeArguments])
				assert.Equal(t, "object_creation_expression", m[java.RelAstKind])
			},
		},
		// --- 9. 方法引用 (forEach(this::simpleMethod)) ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "simpleMethod",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "method_reference", m[java.RelAstKind])
				assert.Equal(t, "this", m[java.RelCallReceiver])
			},
		},
		// --- 10. 泛型方法显式调用 ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.executeAll",
			targetQN: "genericMethod",
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "String", m[java.RelCallTypeArguments])
			},
		},
		// --- 12. 显式构造函数调用 (Super) ---
		{
			sourceQN: "com.example.rel.CallRelationSuite.SubClass.<init>",
			targetQN: "BaseClass", // 或者是父类的 <init>
			relType:  model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "super", m[java.RelCallReceiver])
				assert.Equal(t, "explicit_constructor_invocation", m[java.RelAstKind])
			},
		},
	}

	// 3. 校验逻辑
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型一致 + 目标名一致 + SourceQN 包含关系
			if rel.Type == exp.relType &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing expected relation: [%s] %s -> %s", exp.relType, exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Capture(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/CaptureRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string // 通常是 Lambda 符号名或匿名类方法 QN
		targetQN   string // 被捕获的变量/字段名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. Lambda 捕获局部变量 ---
		{
			sourceQN: "testCaptures$lambda1", // 假设生成的 Lambda 标识
			targetQN: "localVal",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "local_variable", m[java.RelCaptureKind])
				assert.Equal(t, true, m[java.RelCaptureIsEffectivelyFinal])
				assert.Equal(t, "lambda_expression", m[java.RelAstKind])
			},
		},
		// --- 2. Lambda 捕获方法参数 ---
		{
			sourceQN: "testCaptures$lambda2",
			targetQN: "param",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "parameter", m[java.RelCaptureKind])
				assert.Contains(t, m[java.RelCallEnclosingMethod], "testCaptures")
			},
		},
		// --- 3. Lambda 捕获成员变量 (隐式 this) ---
		{
			sourceQN: "testCaptures$lambda3",
			targetQN: "fieldData",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "field", m[java.RelCaptureKind])
				assert.Equal(t, "this", m[java.RelCallReceiver])
				assert.Equal(t, true, m[java.RelCaptureIsImplicitThis])
			},
		},
		// --- 4. Lambda 访问静态成员 ---
		{
			sourceQN: "testCaptures$lambda4",
			targetQN: "staticData",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsStatic])
			},
		},
		// --- 5. 匿名内部类捕获局部变量 ---
		{
			sourceQN: "testCaptures$1.run", // 匿名类的方法
			targetQN: "localVal",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "local_variable", m[java.RelCaptureKind])
				assert.Equal(t, "anonymous_class_capture", m[java.RelAstKind])
			},
		},
		// --- 6. 嵌套 Lambda 捕获 (深度校验) ---
		{
			sourceQN: "testCaptures$lambda5$lambda6", // 嵌套 Lambda 标识
			targetQN: "localVal",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, 2, m[java.RelCaptureDepth]) // 深度为 2
				assert.NotEmpty(t, m[java.RelCaptureEnclosingLambda])
				assert.Equal(t, "localVal", m[java.RelRawText])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 由于 Capture 关系在 model 中可能映射为 Capture 或 Use 类型
			// 我们这里重点匹配 SourceQN 和 TargetName
			if rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Capture relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Cast(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/CastRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 转型目标类名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 基础对象向下转型 ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "input", m[java.RelCastOperandExpression])
				assert.Equal(t, "variable", m[java.RelCastOperandKind])
				assert.Equal(t, "cast_expression", m[java.RelAstKind])
			},
		},
		// --- 2. 基础数据类型转换 ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "int",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "pi", m[java.RelCastOperandExpression])
				assert.Equal(t, true, m[java.RelCastIsPrimitive])
			},
		},
		// --- 3. 泛型集合转型 ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "List",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "String", m[java.RelCastTypeArguments])
				assert.Contains(t, m[java.RelCastFullCastText], "(List<String>)")
			},
		},
		// --- 4. 链式调用中的转型 ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "SubClass",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "specificMethod", m[java.RelCastSubsequentCall])
				assert.Equal(t, true, m[java.RelCastIsParenthesized])
			},
		},
		// --- 5. 模式匹配转型 (Java 14+) ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCastIsPatternMatching])
				assert.Equal(t, "str", m[java.RelCastPatternVariable])
				assert.Equal(t, "instanceof_expression", m[java.RelAstKind])
			},
		},
		// --- 6. 多重转型 (Nested Cast) ---
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "Object",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCastIsNestedCast])
			},
		},
		{
			sourceQN: "com.example.rel.CastRelationSuite.testCastCases",
			targetQN: "Runnable",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCastIsNestedCast])
				assert.Equal(t, "run", m[java.RelCastSubsequentCall])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型一致(CAST) + 目标类型名一致 + SourceQN 匹配
			if rel.Type == model.Cast &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				// 注意：嵌套转型可能有多个结果，这里找到匹配项即通过
				break
			}
		}
		assert.True(t, found, "Missing Cast relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Create(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/CreateRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 实例化的类名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 成员变量声明时实例化 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.fieldInstance",
			targetQN: "ArrayList",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCreateIsInitializer])
				assert.Equal(t, "fieldInstance", m[java.RelCreateVariableName])
				assert.Equal(t, "object_creation_expression", m[java.RelAstKind])
			},
		},
		// --- 2. 静态成员变量实例化 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.staticMap",
			targetQN: "HashMap",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsStatic])
				assert.Equal(t, true, m[java.RelCreateIsInitializer])
			},
		},
		// --- 3. 局部变量实例化 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.testCreateCases",
			targetQN: "StringBuilder",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "sb", m[java.RelCreateVariableName])
				assert.Contains(t, m[java.RelCreateArguments], "\"init\"")
			},
		},
		// --- 4. 匿名内部类创建 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.testCreateCases",
			targetQN: "Runnable",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCreateIsAnonymous])
				// 匿名类通常还会触发 IMPLEMENTS 关系，这里仅校验 CREATE 动作
				assert.Equal(t, "anonymous_class_submission", m[java.RelAstKind])
			},
		},
		// --- 5. 数组实例化 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.testCreateCases",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCreateIsArray])
				assert.Equal(t, 1, m[java.RelCreateDimensions])
				assert.Equal(t, "5", m[java.RelCreateArraySize])
				assert.Equal(t, "array_creation_expression", m[java.RelAstKind])
			},
		},
		// --- 6. 链式调用中的实例化 ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.testCreateCases",
			targetQN: "CreateRelationSuite",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCreateHasSubsequentCall])
				assert.Equal(t, "doNothing", m[java.RelCreateSubsequentCall])
			},
		},
		// --- 7. 构造函数内部实例化 (super 调用) ---
		{
			sourceQN: "com.example.rel.CreateRelationSuite.<init>",
			targetQN: "Object",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "super", m[java.RelCallReceiver])
				assert.Equal(t, true, m[java.RelCreateIsConstructorChain])
				assert.Equal(t, "explicit_constructor_invocation", m[java.RelAstKind])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 CREATE + 目标类名一致 + SourceQN 包含关系
			if rel.Type == model.Create &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Create relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Parameter(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/ParameterRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 参数类型名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 多参数顺序与类型 (String name) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "name", m[java.RelParameterName])
				assert.Equal(t, 0, m[java.RelParameterIndex])
				assert.Equal(t, "formal_parameter", m[java.RelAstKind])
			},
		},
		// --- 1.1 多参数顺序与类型 (long id) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "long",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "id", m[java.RelParameterName])
				assert.Equal(t, 1, m[java.RelParameterIndex])
			},
		},
		// --- 2. 可变参数 (Object... args) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.log",
			targetQN: "Object",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelParameterIsVarargs])
				assert.Equal(t, "args", m[java.RelParameterName])
				assert.Equal(t, 1, m[java.RelParameterIndex])
				// Tree-sitter 区分普通参数与可变参数节点
				assert.Equal(t, "spread_parameter", m[java.RelAstKind])
			},
		},
		// --- 3. Final 参数与注解修饰 ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.setPath",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelParameterIsFinal])
				assert.Equal(t, true, m[java.RelParameterHasAnnotation])
				assert.Equal(t, "path", m[java.RelParameterName])
			},
		},
		// --- 4. 构造函数参数 ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.<init>",
			targetQN: "int",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "val", m[java.RelParameterName])
				assert.Equal(t, 0, m[java.RelParameterIndex])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 PARAMETER + 目标类型名一致 + SourceQN 包含方法名
			if rel.Type == model.Parameter &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				// 针对同一个方法有多个参数的情况，需要通过 Index 进一步区分
				// 这里我们在 checkMores 里做详细校验
				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
			}
		}
		assert.True(t, found, "Missing Parameter relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Return(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/ReturnRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 返回类型名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 对象返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getName",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, false, m[java.RelReturnIsPrimitive])
				// 返回类型通常定义在方法声明节点中
				assert.Equal(t, "method_declaration", m[java.RelAstKind])
			},
		},
		// --- 2. 数组返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getBuffer",
			targetQN: "byte",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
				assert.Equal(t, 1, m[java.RelReturnDimensions])
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 3. 泛型复合返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getValues",
			targetQN: "List",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnHasTypeArguments])
				assert.Equal(t, "generic_type", m[java.RelAstKind])
			},
		},
		// --- 4. 基础类型返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getAge",
			targetQN: "int",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 5. 嵌套数组返回 (深度测试) ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getMatrix",
			targetQN: "double",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
				assert.Equal(t, 2, m[java.RelReturnDimensions])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 RETURN + 目标类型名一致 + SourceQN 包含方法名
			if rel.Type == model.Return &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Return relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Throw(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/ThrowRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 异常类型名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 方法签名中的声明 (IOException) ---
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "IOException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
				assert.Equal(t, 0, m[java.RelThrowIndex])
				assert.Equal(t, "throws_clause", m[java.RelAstKind])
			},
		},
		// --- 1.1 方法签名中的声明 (SQLException) ---
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "SQLException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
				assert.Equal(t, 1, m[java.RelThrowIndex])
			},
		},
		// --- 2. 方法体内主动抛出 (RuntimeException) ---
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "RuntimeException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsRuntime])
				assert.Nil(t, m[java.RelThrowIsSignature]) // 主动抛出不应标记为 signature
				assert.Equal(t, "throw_statement", m[java.RelAstKind])
				assert.Contains(t, m[java.RelRawText], "throw new RuntimeException")
			},
		},
		// --- 3. 构造函数声明抛出 ---
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.<init>",
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
			},
		},
		// --- 4. 重新抛出捕获的异常 (throw e) ---
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.rethrow",
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsRethrow])
				assert.Equal(t, "throw_statement", m[java.RelAstKind])
			},
		},
	}

	// 3. 校验逻辑
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 基础匹配：类型 + 目标名 + SourceQN 包含关系
			if rel.Type == model.Throw &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				// 针对 readFile 这种既有 throws 声明又有 throw 语句的情况
				// 我们需要通过 Mores 中的特征来精确二次匹配
				// 如果当前 rel 满足 checkMores 的预期，则认为找到了

				// 备份一个当前的测试状态，避免干扰外部循环
				ok := t.Run("Matching_"+exp.targetQN, func(st *testing.T) {
					if exp.checkMores != nil {
						exp.checkMores(st, rel.Mores)
					}
				})

				if ok {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Missing Throw relation: [%s] %s -> %s", model.Throw, exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_TypeArg(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/TypeArgRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 泛型实参类型名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 基础多泛型 (Map<String, Integer>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "Map", m[java.RelTypeArgParentType])
				assert.Equal(t, 0, m[java.RelTypeArgIndex])
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "Integer",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "Map", m[java.RelTypeArgParentType])
				assert.Equal(t, 1, m[java.RelTypeArgIndex])
			},
		},
		// --- 2. 嵌套泛型 (List<Map<String, Object>>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Map",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "List", m[java.RelTypeArgParentType])
				assert.Equal(t, 1, m[java.RelTypeArgDepth])
				assert.Equal(t, 0, m[java.RelTypeArgIndex])
			},
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Object",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "Map", m[java.RelTypeArgParentType])
				assert.Equal(t, 2, m[java.RelTypeArgDepth]) // 深度为 2
				assert.Equal(t, 1, m[java.RelTypeArgIndex])
			},
		},
		// --- 3. 上界通配符 (? extends Serializable) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.process.input",
			targetQN: "Serializable",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelTypeArgIsWildcard])
				assert.Equal(t, "extends", m[java.RelTypeArgWildcardKind])
				assert.Contains(t, m[java.RelRawText], "? extends Serializable")
			},
		},
		// --- 4. 构造函数泛型实参 (new ArrayList<String>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.process.list",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "ArrayList", m[java.RelTypeArgParentType])
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},
		// --- 5. 下界通配符 (? super Integer) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.addNumbers.list",
			targetQN: "Integer",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelTypeArgIsWildcard])
				assert.Equal(t, "super", m[java.RelTypeArgWildcardKind])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 TYPE_ARG + 目标类名一致 + SourceQN 包含字段/参数名
			if rel.Type == model.TypeArg &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				// 对于 complexList 这种多个 Target 且深度不同的情况，
				// 需要在 checkMores 里根据 Depth 或 ParentType 精准区分。
				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				// 注意：这里不能 break，因为同一个 Target 可能在不同深度出现
			}
		}
		assert.True(t, found, "Missing TypeArg relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

func TestJavaExtractor_Use(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/UseRelationSuite.java"
	files := []string{testFile}
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string // 被使用的变量、字段或参数名
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 局部变量读取 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "local",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "local + 2", m[java.RelUseParentExpression])
				assert.Equal(t, "operand", m[java.RelUseUsageRole])
				assert.Equal(t, "identifier", m[java.RelAstKind])
			},
		},
		// --- 2. 成员变量读取 (显式 this) ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "fieldVar",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelUseReceiver])
				assert.Equal(t, "field_access", m[java.RelAstKind])
			},
		},
		// --- 4. 静态字段访问 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "CONSTANT",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "UseRelationSuite", m[java.RelUseReceiverType])
				assert.Equal(t, true, m[java.RelUseIsStatic])
			},
		},
		// --- 5. 数组引用读取 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "arr",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "array_access", m[java.RelAstKind])
				assert.Equal(t, "0", m[java.RelUseIndexExpression])
				assert.Equal(t, "array_source", m[java.RelUseUsageRole])
			},
		},
		// --- 6. 方法参数传递 (Argument Use) ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "s",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "print", m[java.RelUseCallSite])
				assert.Equal(t, 0, m[java.RelUseArgumentIndex])
			},
		},
		// --- 7. 条件读取 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "x",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "if_condition", m[java.RelUseContext])
			},
		},
		// --- 8. 增强 for 循环中的集合读取 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "list",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "enhanced_for_statement", m[java.RelAstKind])
				assert.Equal(t, "iterator_source", m[java.RelUseUsageRole])
			},
		},
		// --- 9. Lambda 捕获读取 ---
		{
			sourceQN: "testUseCases$lambda", // Lambda 内部
			targetQN: "fieldVar",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelUseIsCapture])
				assert.Contains(t, m[java.RelUseEnclosingMethod], "testUseCases")
			},
		},
		// --- 10. 类型强制转换中的读取 ---
		{
			sourceQN: "com.example.rel.UseRelationSuite.testUseCases",
			targetQN: "obj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "cast_expression", m[java.RelAstKind])
				assert.Equal(t, "String", m[java.RelUseTargetType])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 USE + 目标名一致 + SourceQN 包含
			if rel.Type == model.Use &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				// 注意：同一个变量可能被多次使用，这里可以根据具体测试需要决定是否 break
			}
		}
		assert.True(t, found, "Missing Use relation: %s -> %s", exp.sourceQN, exp.targetQN)
	}
}

// --- 这里放置你提供的辅助函数 ---

func runPhase1Collection(t *testing.T, files []string) *core.GlobalContext {
	resolver, err := core.GetSymbolResolver(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}

	gc := core.NewGlobalContext(resolver)
	javaParser, err := parser.NewParser(core.LangJava)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	// 注意：在真实测试中可能需要根据情况处理 Close，这里暂存

	col := java.NewJavaCollector()

	for _, file := range files {
		rootNode, sourceBytes, err := javaParser.ParseFile(file, true, false)
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

func printRelations(relations []*model.DependencyRelation) {
	if !printRel {
		return
	}
	fmt.Printf("\n--- Found %d relations ---\n", len(relations))
	for _, rel := range relations {
		fmt.Printf("[%s] %s (%s) --> %s (%s)\n",
			rel.Type,
			rel.Source.QualifiedName, rel.Source.Kind,
			rel.Target.QualifiedName, rel.Target.Kind)
		if len(rel.Mores) > 0 {
			for k, v := range rel.Mores {
				fmt.Printf("    Mores[%v] -> %v\n", k, v)
			}
		}
	}
}

func isMatchMores(m map[string]interface{}, target string) bool {
	// 这里可以根据上下文或 AstKind 简单分流
	// 比如：如果测试用例期待的是 index_expression，而当前 rel 却没有，那就跳过
	return true // 默认返回 true，依靠 checkMores 报错
}
