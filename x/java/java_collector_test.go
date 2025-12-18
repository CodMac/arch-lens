package java_test

import (
	"fmt"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	"github.com/CodMac/go-treesitter-dependency-analyzer/x/java"
	"path/filepath"
	"strings"
	"testing"

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

func TestJavaCollector_MyClass(t *testing.T) {
	// 1. 准备测试源码
	filePath := getTestFilePath("MyClass.java")

	// 2. 解析
	rootNode, sourceBytes, err := getJavaParser(t).ParseFile(filePath, true, true)
	if err != nil {
		t.Fatalf("Failed to parser file: %v", err)
	}

	// 3. 运行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	// 4. 验证 Package Name
	expectedPackage := "com.example.app"
	if fCtx.PackageName != expectedPackage {
		t.Errorf("Expected PackageName %q, got %q", expectedPackage, fCtx.PackageName)
	}

	// 5. 验证 Imports (新特性)
	if importVal, ok := fCtx.Imports["List"]; !ok || importVal != "java.util.List" {
		t.Errorf("Expected import 'List' -> 'java.util.List', got %v", fCtx.Imports["List"])
	}

	// 6. 验证定义集合 (新特性：DefinitionsBySN 是切片，支持同名定义)
	t.Run("Verify Multi-Definition for MyClass", func(t *testing.T) {
		defs := fCtx.DefinitionsBySN["MyClass"]
		if len(defs) != 2 {
			t.Errorf("Expected 2 definitions for 'MyClass' (Class and Constructor), got %d", len(defs))
		}

		hasMethod := false
		hasClass := false
		for _, d := range defs {
			if d.Element.Kind == model.Class {
				hasClass = true
				if d.Element.QualifiedName != "com.example.app.MyClass" {
					t.Errorf("Class QN mismatch: %s", d.Element.QualifiedName)
				}
			}
			if d.Element.Kind == model.Method {
				hasMethod = true
				if d.Element.QualifiedName != "com.example.app.MyClass.MyClass" {
					t.Errorf("Constructor QN mismatch: %s", d.Element.QualifiedName)
				}
			}
		}
		if !hasClass || !hasMethod {
			t.Errorf("Missing Class or Constructor in MyClass definitions")
		}
	})

	t.Run("Verify Enum and Constants", func(t *testing.T) {
		// 验证 Enum
		enumDefs := fCtx.DefinitionsBySN["Status"]
		if len(enumDefs) == 0 || enumDefs[0].Element.Kind != model.Enum {
			t.Errorf("Enum 'Status' not found correctly")
		}

		// 验证 Enum Constant
		activeDefs := fCtx.DefinitionsBySN["ACTIVE"]
		if len(activeDefs) == 0 || activeDefs[0].Element.Kind != model.EnumConstant {
			t.Errorf("EnumConstant 'ACTIVE' not found")
		}
		if activeDefs[0].ParentQN != "com.example.app.Status" {
			t.Errorf("EnumConstant parent mismatch, got %s", activeDefs[0].ParentQN)
		}
	})

	t.Run("Verify Extra Information", func(t *testing.T) {
		// 检查 counter 是否有注解
		counterDefs := fCtx.DefinitionsBySN["counter"]
		if len(counterDefs) > 0 {
			elem := counterDefs[0].Element
			if elem.Extra == nil || len(elem.Extra.Annotations) == 0 {
				t.Errorf("Field 'counter' missing annotations")
			} else if !strings.Contains(elem.Extra.Annotations[0], "MyFieldAnnotation") {
				t.Errorf("Unexpected annotation: %s", elem.Extra.Annotations[0])
			}
		}
	})

	fmt.Println("Java Collector test completed successfully.")
}
