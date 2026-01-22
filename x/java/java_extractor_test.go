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
	testFile := "testdata/com/example/rel/AnnotationRelationSuite.java"
	files := []string{testFile}

	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

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
			},
		},
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite",
			targetQN:   "SuppressWarnings",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "TYPE", m[java.RelAnnotationTarget])
			},
		},
		// --- 2. 字段注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.id",
			targetQN:   "Id",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "FIELD", m[java.RelAnnotationTarget])
			},
		},
		// --- 3. 方法注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.save(String)",
			targetQN:   "Transactional",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "METHOD", m[java.RelAnnotationTarget])
				// 注意：RelAnnotationParams 已移至 Extended，此处不再断言
			},
		},
		// --- 4. 局部变量注解 ---
		{
			relType:    model.Annotation,
			sourceQN:   "com.example.rel.AnnotationRelationSuite.save(String).local",
			targetQN:   "NonEmpty",
			targetKind: model.KAnnotation,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "LOCAL_VARIABLE", m[java.RelAnnotationTarget])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			if rel.Type == exp.relType &&
				rel.Target.Name == exp.targetQN &&
				strings.HasSuffix(rel.Source.QualifiedName, exp.sourceQN) {

				found = true
				assert.Equal(t, exp.targetKind, rel.Target.Kind)
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing expected relation: [%s] %s -> %s", exp.relType, exp.sourceQN, exp.targetQN)
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

	printRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string               // Source 节点的 QN 片段
		targetName string               // Target 节点的名称 (Short Name)
		relType    model.DependencyType // 关系类型
		value      string               // 对应 RelRawText 的精确定位
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "this", m[java.RelCallReceiver])
				assert.Equal(t, false, m[java.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "staticMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsStatic])
				assert.Equal(t, "CallRelationSuite", m[java.RelCallReceiverType])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "currentTimeMillis",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "System", m[java.RelCallReceiver])
				assert.Equal(t, true, m[java.RelCallIsStatic])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "add",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsChained])
			},
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "ArrayList",
			relType:    model.Create, // 确认 Create 逻辑存在
		},
		{
			sourceQN:   "CallRelationSuite.executeAll()",
			targetName: "ArrayList",
			relType:    model.Call, // 采纳建议：同时也存在 CALL 构造函数
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelCallIsConstructor])
			},
		},
		{
			sourceQN:   "lambda$1",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[java.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "anonymousClass$1.run()",
			targetName: "simpleMethod",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "com.example.rel.CallRelationSuite.executeAll()", m[java.RelCallEnclosingMethod])
			},
		},
		{
			sourceQN:   "SubClass.SubClass()",
			targetName: "super",
			relType:    model.Call,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "explicit_constructor_invocation", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelCallIsConstructor])
			},
		},
	}

	// 3. 执行断言
	for _, exp := range expectedRels {
		t.Run(fmt.Sprintf("%s_to_%s", exp.relType, exp.targetName), func(t *testing.T) {
			found := false
			for _, rel := range allRelations {
				if rel.Type == exp.relType &&
					strings.Contains(rel.Target.Name, exp.targetName) &&
					strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

					found = true
					if exp.checkMores != nil {
						exp.checkMores(t, rel.Mores)
					}
					break
				}
			}
			assert.True(t, found, "Missing: [%s] Source:%s -> Target:%s",
				exp.relType, exp.sourceQN, exp.targetName)
		})
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

func TestJavaExtractor_Assign(t *testing.T) {
	// 1. 准备与提取
	testFile := "testdata/com/example/rel/AssignRelationSuite.java"
	files := []string{testFile}

	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceMatch string // 匹配 Source.QualifiedName
		targetMatch string // 匹配 Target.Name
		matchMores  func(m map[string]interface{}) bool
		checkMores  func(t *testing.T, mores map[string]interface{})
	}{
		// 1. 字段声明初始化
		{
			sourceMatch: "AssignRelationSuite.count",
			targetMatch: "count",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "0", m[java.RelAssignValueExpression])
			},
		},
		// 2. 静态块赋值
		{
			sourceMatch: "$static$1",
			targetMatch: "status",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "\"INIT\"", m[java.RelAssignValueExpression])
			},
		},
		// 3. 局部变量基础赋值
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "local",
			matchMores: func(m map[string]interface{}) bool {
				return m[java.RelAssignIsInitializer] == true
			},
		},
		// 6. 链式赋值 (b)
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "b",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "c = 50", m[java.RelAssignValueExpression])
			},
		},
		// 8. 数组元素赋值 (Target 应该是数组变量名)
		{
			sourceMatch: "testAssignments(int)",
			targetMatch: "arr",
			matchMores: func(m map[string]interface{}) bool {
				return strings.Contains(fmt.Sprintf("%v", m[java.RelRawText]), "arr[0]")
			},
		},
		// 9. Lambda 内部赋值
		{
			sourceMatch: "lambda$1",
			targetMatch: "count",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "300", m[java.RelAssignValueExpression])
			},
		},
	}

	// 执行匹配循环
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 1. 必须是 Assign 关系
			if rel.Type != model.Assign {
				continue
			}

			// 2. 匹配 Source (支持 QN 后缀匹配)
			sourceOk := strings.Contains(rel.Source.QualifiedName, exp.sourceMatch)

			// 3. 匹配 Target (支持短名或 QN 匹配)
			// 关键修复：同时检查 Name 和 QualifiedName
			targetOk := rel.Target.Name == exp.targetMatch ||
				strings.HasSuffix(rel.Target.QualifiedName, "."+exp.targetMatch)

			if sourceOk && targetOk {
				if exp.matchMores != nil && !exp.matchMores(rel.Mores) {
					continue
				}
				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Assign: %s -> %s", exp.sourceMatch, exp.targetMatch)
	}
}

func TestJavaExtractor_AssignClass(t *testing.T) {
	testFile := "testdata/com/example/rel/AssignRelationForClassSuite.java"
	files := []string{testFile}

	// 假设 runPhase1Collection 已经处理了符号定义
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(allRelations)

	expectedRels := []struct {
		sourceQN   string
		targetName string
		value      string // 新增：用于精确定位
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		{
			sourceQN:   "testClassAssignments",
			targetName: "list",
			value:      "new ArrayList<>()",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		{
			sourceQN:   "testClassAssignments",
			targetName: "name",
			value:      "\"Hello\"",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "assignment_expression", m[java.RelAstKind])
			},
		},
		{
			sourceQN:   "testClassAssignments",
			targetName: "data",
			value:      "new DataNode()", // 匹配第一处赋值
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "variable_declarator", m[java.RelAstKind])
			},
		},
		{
			sourceQN:   "testClassAssignments",
			targetName: "data",
			value:      "null", // 匹配第二处赋值
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "assignment_expression", m[java.RelAstKind])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 增加 ValueExpression 的匹配校验
			relValue, _ := rel.Mores[java.RelAssignValueExpression].(string)

			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.Name == exp.targetName &&
				relValue == exp.value { // 精确匹配

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Assign: %s -> %s (value: %s)", exp.sourceQN, exp.targetName, exp.value)
	}
}

func TestJavaExtractor_AssignDataFlow(t *testing.T) {
	// 1. 准备测试文件路径（注意文件名需与 testdata 目录一致）
	testFile := "testdata/com/example/rel/AssignRelationForDataFlow.java"
	files := []string{testFile}

	// 2. 运行符号收集与提取逻辑
	gCtx := runPhase1Collection(t, files)
	extractor := java.NewJavaExtractor()
	allRelations, err := extractor.Extract(testFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// 打印结果便于调试
	printRelations(allRelations)

	// 3. 定义预期关系
	expectedRels := []struct {
		sourceQN   string
		targetName string
		value      string // 用于精确定位具体的赋值语句
		checkMores func(t *testing.T, mores map[string]interface{})
	}{
		// --- 1. 常量赋值 (this.data = "CONST") ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "data",
			value:      "\"CONST\"",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "assignment_expression", m[java.RelAstKind])
				assert.Equal(t, "=", m[java.RelAssignOperator])
			},
		},
		// --- 2. 返回值流向 (Object localObj = fetch()) ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "localObj",
			value:      "fetch()",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "variable_declarator", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
			},
		},
		// --- 3. 转换流向 (String msg = (String) localObj) ---
		{
			sourceQN:   "com.example.rel.AssignRelationForDataFlow.testDataFlow",
			targetName: "msg",
			value:      "(String) localObj",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "variable_declarator", m[java.RelAstKind])
				assert.Equal(t, true, m[java.RelAssignIsInitializer])
				assert.Equal(t, "msg", m[java.RelAssignTargetName])
			},
		},
	}

	// 4. 执行匹配与验证逻辑
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 获取当前关系的 ValueExpression 以便精确定位
			relValue, _ := rel.Mores[java.RelAssignValueExpression].(string)

			// 匹配 ASSIGN 类型，且 Source QN、Target Name 和 Value 对齐
			if rel.Type == model.Assign &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				rel.Target.Name == exp.targetName &&
				relValue == exp.value {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Data Flow relation: %s -> %s (value: %s)",
			exp.sourceQN, exp.targetName, exp.value)
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

func TestJavaExtractor_Parameter(t *testing.T) {
	testFile := "testdata/com/example/rel/ParameterRelationSuite.java"
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
		index      int // 显式提取 Index 以便在多参数场景下精准匹配
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 多参数顺序与类型 (String name) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "name", m[java.RelParameterName])
				assert.Equal(t, 0, m[java.RelParameterIndex])
			},
		},
		// --- 1.1 多参数顺序与类型 (long id) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.update",
			targetQN: "long",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "id", m[java.RelParameterName])
				assert.Equal(t, 1, m[java.RelParameterIndex])
			},
		},
		// --- 2. 可变参数 (Object... args) ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.log",
			targetQN: "Object",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelParameterIsVarargs])
				assert.Equal(t, "args", m[java.RelParameterName])
			},
		},
		// --- 3. Final 参数与注解修饰 ---
		{
			sourceQN: "com.example.rel.ParameterRelationSuite.setPath",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "path", m[java.RelParameterName])
			},
		},
		// --- 4. 构造函数参数 ---
		{
			sourceQN: "ParameterRelationSuite", // 兼容 <init> 或 类名
			targetQN: "int",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "val", m[java.RelParameterName])
			},
		},
	}

	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 匹配原则：类型为 PARAMETER + 目标类型名一致 + SourceQN 匹配 + Index 一致
			relIndex, _ := rel.Mores[java.RelParameterIndex].(int)

			if rel.Type == model.Parameter &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				relIndex == exp.index {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing Parameter relation: %s -> %s (index %d)", exp.sourceQN, exp.targetQN, exp.index)
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

	printRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 对象返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getName",
			targetQN: "String",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 默认不标记 is_primitive 时，Extractor 应根据类型识别并填充
				assert.Equal(t, false, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 2. 数组返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getBuffer",
			targetQN: "byte",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 3. 泛型复合返回 ---
		{
			sourceQN:   "com.example.rel.ReturnRelationSuite.getValues",
			targetQN:   "List",
			checkMores: func(t *testing.T, m map[string]interface{}) {},
		},
		// --- 4. 基础类型返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getAge",
			targetQN: "int",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsPrimitive])
			},
		},
		// --- 5. 嵌套数组返回 ---
		{
			sourceQN: "com.example.rel.ReturnRelationSuite.getMatrix",
			targetQN: "double",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelReturnIsArray])
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
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

	printRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "IOException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
				assert.Equal(t, 0, m[java.RelThrowIndex])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "SQLException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
				assert.Equal(t, 1, m[java.RelThrowIndex])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.readFile",
			targetQN: "RuntimeException",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				isSig, _ := m[java.RelThrowIsSignature].(bool)
				assert.False(t, isSig)
				assert.Contains(t, m[java.RelRawText], "throw new RuntimeException")
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.ThrowRelationSuite", // 改掉 <init>
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, true, m[java.RelThrowIsSignature])
			},
		},
		{
			sourceQN: "com.example.rel.ThrowRelationSuite.rethrow",
			targetQN: "Exception",
			checkMores: func(t *testing.T, m map[string]interface{}) {
				// 重新抛出暂无特殊标记
			},
		},
	}

	// 3. 校验逻辑 (修正 Unused Variable 问题)
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 基础条件匹配
			if rel.Type == model.Throw &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) {

				// 如果有 checkMores，需要确保当前这个 rel 满足 mores 里的特定条件
				// (防止在 readFile 里把 IOException 错认成 SQLException)
				if exp.checkMores != nil {
					// 使用匿名测试函数进行判定
					isCurrentMatch := t.Run("SubCheck", func(st *testing.T) {
						exp.checkMores(st, rel.Mores)
					})

					if !isCurrentMatch {
						continue // 当前 rel 属性不匹配，去找下一个
					}
				}

				found = true
				break
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

	printRelations(allRelations)

	// 2. 定义断言数据集
	expectedRels := []struct {
		sourceQN   string
		targetQN   string
		index      int
		checkMores func(t *testing.T, m map[string]interface{})
	}{
		// --- 1. 基础多泛型 (Map<String, Integer>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, 0, m[java.RelTypeArgIndex])
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.map",
			targetQN: "Integer",
			index:    1,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, 1, m[java.RelTypeArgIndex])
			},
		},

		// --- 2. 嵌套泛型 (List<Map<String, Object>>) ---
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Map",
			index:    0,
		},
		{
			sourceQN: "com.example.rel.TypeArgRelationSuite.complexList",
			targetQN: "Object",
			index:    1, // 对应 Map<String, Object> 的第二个参数
		},

		// --- 3. 上界通配符 (? extends Serializable) ---
		{
			// 使用方法名和参数名片段，兼容 "process(List).input"
			sourceQN: "TypeArgRelationSuite.process",
			targetQN: "Serializable",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Contains(t, m[java.RelRawText], "? extends Serializable")
			},
		},

		// --- 4. 构造函数泛型实参 (new ArrayList<String>) ---
		{
			sourceQN: "TypeArgRelationSuite.process",
			targetQN: "String",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Equal(t, "type_arguments", m[java.RelAstKind])
			},
		},

		// --- 5. 下界通配符 (? super Integer) ---
		{
			sourceQN: "TypeArgRelationSuite.addNumbers",
			targetQN: "Integer",
			index:    0,
			checkMores: func(t *testing.T, m map[string]interface{}) {
				assert.Contains(t, m[java.RelRawText], "? super Integer")
			},
		},
	}

	// 3. 执行匹配断言
	for _, exp := range expectedRels {
		found := false
		for _, rel := range allRelations {
			// 获取实际的 Index
			relIndex, _ := rel.Mores[java.RelTypeArgIndex].(int)

			// 匹配原则：类型为 TYPE_ARG + 目标类名一致 + SourceQN 包含关键词 + Index 一致
			if rel.Type == model.TypeArg &&
				rel.Target.Name == exp.targetQN &&
				strings.Contains(rel.Source.QualifiedName, exp.sourceQN) &&
				relIndex == exp.index {

				found = true
				if exp.checkMores != nil {
					exp.checkMores(t, rel.Mores)
				}
				break
			}
		}
		assert.True(t, found, "Missing TypeArg: %s -> %s (index %d)", exp.sourceQN, exp.targetQN, exp.index)
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
