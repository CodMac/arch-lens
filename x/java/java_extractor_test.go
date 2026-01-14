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
	"github.com/stretchr/testify/assert"
)

// 辅助函数：解析并收集定义 (Phase 1)
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
	defer javaParser.Close()

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

const printRel = true

func printRelations(relations []*model.DependencyRelation) {
	if !printRel {
		return
	}

	fmt.Printf("Found relation: [DependencyType] -> source(Kind::QualifiedName)==>target(Kind::QualifiedName)\n")
	for _, rel := range relations {
		fmt.Printf("Found relation: [%s] -> source(%s::%s)==>target(%s::%s)\n", rel.Type, rel.Source.Kind, rel.Source.QualifiedName, rel.Target.Kind, rel.Target.QualifiedName)
	}
}

func TestJavaExtractor_AbstractBaseEntity_DeepValidation(t *testing.T) {
	// 1. 准备环境
	targetFile := getTestFilePath(filepath.Join("com", "example", "base", "AbstractBaseEntity.java"))
	gCtx := runPhase1Collection(t, []string{targetFile})

	ext := java.NewJavaExtractor()
	relations, err := ext.Extract(targetFile, gCtx)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	printRelations(relations)

	// --- 1. 验证层级关系 (CONTAIN) ---
	t.Run("Verify Comprehensive Hierarchy", func(t *testing.T) {
		hasPkgToClass := false
		hasClassToInner := false
		hasClassToField := false
		hasClassToMethod := false

		for _, rel := range relations {
			if rel.Type != model.Contain {
				continue
			}
			// Package -> Class
			if rel.Source.Kind == model.Package && rel.Target.Name == "AbstractBaseEntity" {
				hasPkgToClass = true
			}
			// Class -> InnerClass
			if rel.Source.Name == "AbstractBaseEntity" && rel.Target.Name == "EntityMeta" {
				hasClassToInner = true
				assert.Equal(t, model.Class, rel.Target.Kind)
			}
			// Class -> Field
			if rel.Source.Name == "AbstractBaseEntity" && rel.Target.Name == "id" {
				hasClassToField = true
				assert.Equal(t, model.Field, rel.Target.Kind)
			}
			// Class -> Method
			if rel.Source.Name == "AbstractBaseEntity" && rel.Target.Name == "getId" {
				hasClassToMethod = true
				assert.Equal(t, model.Method, rel.Target.Kind)
			}
		}
		assert.True(t, hasPkgToClass, "Missing: Package/File -> Class")
		assert.True(t, hasClassToInner, "Missing: Class -> InnerClass")
		assert.True(t, hasClassToField, "Missing: Class -> Field")
		assert.True(t, hasClassToMethod, "Missing: Class -> Method")
	})

	// --- 2. 验证方法维度关系 (RETURN, PARAMETER, USE, ASSIGN) ---
	t.Run("Verify Method Behaviors", func(t *testing.T) {
		foundReturn := false
		foundParam := false
		foundUse := false
		foundAssign := false

		for _, rel := range relations {
			// RETURN: getId() -> ID
			if rel.Type == model.Return && rel.Source.Name == "getId" && rel.Target.Name == "ID" {
				foundReturn = true
			}
			// PARAMETER: setId(ID id) -> ID
			if rel.Type == model.Parameter && rel.Source.Name == "setId" && rel.Target.Name == "ID" {
				foundParam = true
			}
			// USE: getId() { return id; } -> 这里的 Source 是方法，Target 是字段
			if rel.Type == model.Use && rel.Source.Name == "getId" && rel.Target.Name == "id" {
				foundUse = true
			}
			// ASSIGN: this.id = id -> 这里的 Source 是方法/语句，Target 是被赋值的字段 id
			if rel.Type == model.Assign && rel.Target.Name == "id" {
				// 校验赋值动作是否发生在 setId 方法内
				if strings.Contains(rel.Source.QualifiedName, "setId") {
					foundAssign = true
				}
			}
		}
		assert.True(t, foundReturn, "Should detect Return relation")
		assert.True(t, foundParam, "Should detect Parameter relation")
		assert.True(t, foundUse, "Should detect Use (read) relation in getId")
		assert.True(t, foundAssign, "Should detect Assign (write) relation in setId")
	})

	// --- 3. 验证实现关系 (IMPLEMENT) ---
	t.Run("Verify Implementation", func(t *testing.T) {
		found := false
		for _, rel := range relations {
			if rel.Type == model.Implement && rel.Source.Name == "AbstractBaseEntity" {
				if rel.Target.Name == "Serializable" {
					found = true
					assert.Contains(t, rel.Target.QualifiedName, "Serializable")
				}
			}
		}
		assert.True(t, found, "AbstractBaseEntity should implement Serializable")
	})

	// --- 4. 验证自定义 Mores 常量信息 ---
	t.Run("Verify Custom Mores Metadata", func(t *testing.T) {
		for _, rel := range relations {
			// 对动作类关系（如 Use/Assign/Call）验证元数据
			if rel.Type == model.Use || rel.Type == model.Assign {
				assert.NotEmpty(t, rel.Mores[java.RelRawText], "Mores should have RawText")
				assert.NotEmpty(t, rel.Mores[java.RelContext], "Mores should have AST Kind")

				// 验证特定的 AST Kind，例如 Assignment 应该是 assignment_expression
				if rel.Type == model.Assign {
					assert.Equal(t, "assignment_expression", rel.Mores[java.RelContext])
				}
			}
		}
	})
}
