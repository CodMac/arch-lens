package java_test

import (
	"strings"
	"testing"

	"github.com/CodMac/go-treesitter-dependency-analyzer/extractor"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java" // 触发 init() 注册
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

func TestJavaExtractor_Extract_Integration(t *testing.T) {
	// 1. 准备测试文件路径
	// 我们需要加载所有相关文件以构建完整的 GlobalContext，
	// 这样 Extractor 才能正确解析跨文件引用 (如 UserService 使用 User)。
	testFiles := []string{
		getTestFilePath("User.java"),
		getTestFilePath("UserService.java"),
		getTestFilePath("ErrorCode.java"),
		getTestFilePath("NotificationException.java"),
	}

	// 2. Phase 1: 构建全局上下文 (Symbol Table)
	gCtx := runPhase1Collection(t, testFiles)

	// 3. Phase 2: 针对 UserService.java 运行 Extractor
	// UserService.java 是依赖关系最丰富的文件
	targetFile := getTestFilePath("UserService.java")

	// 获取 Extractor
	ext, err := extractor.GetExtractor(model.LangJava)
	if err != nil {
		t.Fatalf("Failed to get Java extractor: %v", err)
	}

	// 获取 UserService 的 RootNode (Phase 1 已经解析过，理论上应该复用，这里为了模拟流程重新获取或从 gCtx 拿)
	// 在真实 Processor 中是直接传递 RootNode 的。这里我们从 gCtx 获取。
	fCtx := gCtx.FileContexts[targetFile]
	if fCtx == nil {
		t.Fatalf("FileContext not found for %s", targetFile)
	}

	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extractor failed: %v", err)
	}

	// 4. 验证依赖关系
	// 我们将检查 relations 切片中是否存在特定的关键依赖
	validateRelations(t, relations)
}

func validateRelations(t *testing.T, relations []*model.DependencyRelation) {
	// 定义期望存在的依赖关系特征
	expectations := []struct {
		relType    model.DependencyType
		targetName string
		targetKind model.ElementKind
		found      bool
	}{
		// --- Import ---
		{model.Import, "com.example.model.User", model.Package, false},
		{model.Import, "com.example.model.ErrorCode", model.Package, false},

		// --- Structure (Extend / Implement / Annotation / Contain) ---
		{model.Implement, "DataService", model.Interface, false},  // UserService implements DataService
		{model.Annotation, "Service", model.KAnnotation, false},   // @Service
		{model.Annotation, "Autowired", model.KAnnotation, false}, // @Autowired
		{model.Contain, "findById", model.Method, false},          // UserService contains findById

		// --- Actions (Call / Create / Use / Cast) ---
		// Call: repository.findOne(id)
		{model.Call, "findOne", model.Method, false},
		// Create: new User(name)
		{model.Create, "User", model.Class, false},
		// Create: new NotificationException(...)
		{model.Create, "NotificationException", model.Class, false},
		// Use: ErrorCode.USER_NOT_FOUND (Field Access / Enum Constant use)
		{model.Use, "USER_NOT_FOUND", model.Field, false}, // Tree-sitter query 可能会将其识别为 field access
		// Cast: (User) ...
		{model.Cast, "User", model.Type, false},
	}

	// 遍历实际结果进行匹配
	for _, rel := range relations {
		for i := range expectations {
			exp := &expectations[i]
			if exp.found {
				continue
			}

			// 简单的匹配逻辑：类型匹配且名称包含
			// 注意：rel.Target.Name 可能是短名，QualifiedName 可能是全名
			// 这里主要匹配 Name 或 QualifiedName
			nameMatch := strings.Contains(rel.Target.Name, exp.targetName) ||
				strings.Contains(rel.Target.QualifiedName, exp.targetName)

			if rel.Type == exp.relType && nameMatch {
				// 如果期望了 Kind，则校验 Kind
				if exp.targetKind != "" && rel.Target.Kind != exp.targetKind {
					// 对于 USE 关系，Tree-sitter 有时将 EnumConstant 识别为 Field，这里做宽容处理
					if rel.Type == model.Use && (rel.Target.Kind == model.Field || rel.Target.Kind == model.EnumConstant) {
						// Pass
					} else {
						continue
					}
				}
				exp.found = true
				t.Logf("✅ Found expected relation: [%s] -> %s (%s)", rel.Type, rel.Target.Name, rel.Target.Kind)
			}
		}
	}

	// 检查是否有未找到的期望
	for _, exp := range expectations {
		if !exp.found {
			t.Errorf("❌ Missing expected relation: Type=[%s] Target=[%s] Kind=[%s]", exp.relType, exp.targetName, exp.targetKind)
		}
	}

	// 额外测试：验证 ErrorCode.java 的 Enum Constant 包含关系
	// 这部分逻辑在 Extractor 的 handleDefinitionAndStructureRelations 中
	// 我们不需要重新运行 extractor，只需要写一个新的小测试或者在这里扩展逻辑，
	// 为了清晰，我们假定上面的测试覆盖了主要流程。
}

func TestJavaExtractor_EnumStructure(t *testing.T) {
	// 单独测试 ErrorCode.java 的结构提取
	file := getTestFilePath("ErrorCode.java")
	gCtx := runPhase1Collection(t, []string{file})

	ext, _ := extractor.GetExtractor(model.LangJava)
	relations, err := ext.Extract(file, gCtx)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	foundUserNotFound := false
	foundNameEmpty := false

	for _, rel := range relations {
		if rel.Type == model.Contain && rel.Target.Kind == model.EnumConstant {
			if rel.Target.Name == "USER_NOT_FOUND" {
				foundUserNotFound = true
			}
			if rel.Target.Name == "NAME_EMPTY" {
				foundNameEmpty = true
			}
		}
	}

	if !foundUserNotFound {
		t.Error("Failed to extract Enum Constant 'USER_NOT_FOUND' in ErrorCode")
	}
	if !foundNameEmpty {
		t.Error("Failed to extract Enum Constant 'NAME_EMPTY' in ErrorCode")
	}
}
