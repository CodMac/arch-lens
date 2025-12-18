package java

import (
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Collector 实现了 collector.Collector 接口
type Collector struct{}

func NewJavaCollector() *Collector {
	return &Collector{}
}

// CollectDefinitions 实现了 extractor.DefinitionCollector 接口
func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*model.FileContext, error) {
	fCtx := model.NewFileContext(filePath, rootNode, sourceBytes)

	// 1. 处理 Package Name 和 Import Declarations
	c.processTopLevelDeclarations(fCtx)

	// 2. 递归收集定义
	// 初始 QN Stack 只包含 PackageName（如果存在）
	initialQN := ""
	if fCtx.PackageName != "" {
		initialQN = fCtx.PackageName
	}

	if err := c.collectDefinitionsRecursive(fCtx.RootNode, fCtx, initialQN); err != nil {
		return nil, err
	}

	return fCtx, nil
}

// processTopLevelDeclarations 处理包名和导入语句
func (c *Collector) processTopLevelDeclarations(fCtx *model.FileContext) {
	// 遍历 program 的直属子节点
	for i := 0; i < int(fCtx.RootNode.ChildCount()); i++ {
		child := fCtx.RootNode.Child(uint(i))
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "package_declaration":
			// 提取包名 (例如: package com.example.app;)
			if pkgNameNode := child.NamedChild(0); pkgNameNode != nil {
				fCtx.PackageName = getNodeContent(pkgNameNode, *fCtx.SourceBytes)
			}
		case "import_declaration":
			// 提取导入 (例如: import java.util.List;)
			// 注意：Java 也有 static import 和 wildcard import (*)，这里先处理基础的全限定名导入
			if importNameNode := child.NamedChild(0); importNameNode != nil {
				fullImportPath := getNodeContent(importNameNode, *fCtx.SourceBytes)
				// 获取短名称作为 Key (例如: List)
				parts := strings.Split(fullImportPath, ".")
				if len(parts) > 0 {
					shortName := parts[len(parts)-1]
					// 填充 model/context.go 中新增的 Imports 映射
					fCtx.Imports[shortName] = fullImportPath
				}
			}
		}
	}
}

// collectDefinitionsRecursive 使用递归来简化 QN 栈的管理。
func (c *Collector) collectDefinitionsRecursive(node *sitter.Node, fCtx *model.FileContext, currentQNPrefix string) error {
	// 检查当前节点是否是一个定义
	if elem, kind := getDefinitionElement(node, fCtx.SourceBytes, fCtx.FilePath); elem != nil {
		// 1. 构造 Qualified Name
		parentQN := currentQNPrefix
		elem.QualifiedName = model.BuildQualifiedName(parentQN, elem.Name)

		// 2. 注册定义 (内部会调用 fc.DefinitionsBySN[elem.Name] = append(...))
		fCtx.AddDefinition(elem, parentQN)

		// 3. 更新 QN 前缀
		// 对于 Class/Interface/Enum/Method/Annotation，它们是容器
		if kind == model.Class || kind == model.Interface || kind == model.Enum || kind == model.Method {
			currentQNPrefix = elem.QualifiedName
		}
	}

	// 递归遍历子节点
	cursor := node.Walk()
	defer cursor.Close()

	if cursor.GotoFirstChild() {
		for {
			if err := c.collectDefinitionsRecursive(cursor.Node(), fCtx, currentQNPrefix); err != nil {
				return err
			}
			if !cursor.GotoNextSibling() {
				break
			}
		}
	}

	return nil
}

// extractLocation 从 sitter.Node 中提取位置信息
func extractLocation(n *sitter.Node, filePath string) *model.Location {
	if n == nil {
		return nil
	}
	return &model.Location{
		FilePath:    filePath,
		StartLine:   int(n.StartPosition().Row) + 1,
		EndLine:     int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column),
		EndColumn:   int(n.EndPosition().Column),
	}
}

// extractModifiersAndAnnotations 提取修饰符和注解
func extractModifiersAndAnnotations(n *sitter.Node, sourceBytes []byte) ([]string, []string) {
	var modifiers []string
	var annotations []string

	var modifiersNode *sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(uint(i))
		if child.Kind() == "modifiers" {
			modifiersNode = child
			break
		}
	}

	if modifiersNode == nil {
		return modifiers, annotations
	}

	// 提取关键字修饰符
	content := getNodeContent(modifiersNode, sourceBytes)
	fields := strings.Fields(content)
	for _, modifier := range fields {
		trimSpace := strings.TrimSpace(modifier)
		if isKeywordModifier(trimSpace) {
			modifiers = append(modifiers, trimSpace)
		}
	}

	// 提取注解
	for i := 0; i < int(modifiersNode.ChildCount()); i++ {
		child := modifiersNode.Child(uint(i))
		if strings.Contains(child.Kind(), "annotation") {
			annotations = append(annotations, getNodeContent(child, sourceBytes))
		}
	}

	return modifiers, annotations
}

func isKeywordModifier(s string) bool {
	switch s {
	case "public", "private", "protected", "static", "final", "abstract",
		"synchronized", "transient", "volatile", "default", "native", "strictfp":
		return true
	}
	return false
}

func extractMethodSignature(node *sitter.Node, sourceBytes []byte) string {
	var parts []string
	modifiers, _ := extractModifiersAndAnnotations(node, sourceBytes)
	if len(modifiers) > 0 {
		parts = append(parts, strings.Join(modifiers, " "))
	}
	if node.Kind() == "method_declaration" {
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			parts = append(parts, getNodeContent(typeNode, sourceBytes))
		}
	}
	name := ""
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		name = getNodeContent(nameNode, sourceBytes)
	} else if node.Kind() == "constructor_declaration" {
		if parent := node.Parent(); parent != nil {
			if classNameNode := parent.ChildByFieldName("name"); classNameNode != nil {
				name = getNodeContent(classNameNode, sourceBytes)
			}
		}
	}
	if name != "" {
		parts = append(parts, name)
	}
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		parts = append(parts, getNodeContent(paramsNode, sourceBytes))
	} else if node.Kind() == "method_declaration" || node.Kind() == "constructor_declaration" {
		parts = append(parts, "()")
	}
	if throwsNode := node.ChildByFieldName("throws"); throwsNode != nil {
		parts = append(parts, getNodeContent(throwsNode, sourceBytes))
	}
	return strings.Join(parts, " ")
}

func getDefinitionElement(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	if node == nil {
		return nil, ""
	}

	var elem *model.CodeElement
	var kind model.ElementKind
	var nameNode *sitter.Node

	switch node.Kind() {
	case "class_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Class
	case "interface_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Interface
	case "enum_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Enum
	case "enum_constant":
		nameNode = node.ChildByFieldName("name")
		kind = model.EnumConstant
	case "method_declaration", "constructor_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Method
	case "field_declaration":
		if vNode := findNamedChildOfType(node, "variable_declarator"); vNode != nil {
			nameNode = vNode.ChildByFieldName("name")
			kind = model.Field
		}
	case "annotation_type_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Interface
	}

	if nameNode == nil && (kind == model.Class || kind == model.Interface || kind == model.Enum || kind == model.Field) {
		return nil, ""
	}

	name := ""
	if nameNode != nil {
		name = getNodeContent(nameNode, *sourceBytes)
	} else if node.Kind() == "constructor_declaration" {
		if parent := node.Parent(); parent != nil && (parent.Kind() == "class_declaration" || parent.Kind() == "enum_declaration") {
			if classNameNode := parent.ChildByFieldName("name"); classNameNode != nil {
				name = getNodeContent(classNameNode, *sourceBytes)
			}
		}
		if name == "" {
			name = "Constructor"
		}
	}

	if name == "" {
		return nil, ""
	}

	elem = &model.CodeElement{
		Kind: kind,
		Name: name,
		Path: filePath,
	}
	elem.Location = extractLocation(node, filePath)

	extra := &model.ElementExtra{}
	modifiers, annotations := extractModifiersAndAnnotations(node, *sourceBytes)
	if len(modifiers) > 0 {
		extra.Modifiers = modifiers
	}
	if len(annotations) > 0 {
		extra.Annotations = annotations
	}

	switch kind {
	case model.Class, model.Interface, model.Enum:
		classExtra := &model.ClassExtra{}
		if node.Kind() == "class_declaration" {
			if extendsNode := node.ChildByFieldName("superclass"); extendsNode != nil {
				if typeNode := extendsNode.NamedChild(0); typeNode != nil {
					classExtra.SuperClass = getNodeContent(typeNode, *sourceBytes)
				}
			}
			if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
				if typeListNode := interfacesNode.NamedChild(0); typeListNode != nil {
					for i := 0; i < int(typeListNode.NamedChildCount()); i++ {
						classExtra.ImplementedInterfaces = append(classExtra.ImplementedInterfaces, getNodeContent(typeListNode.NamedChild(uint(i)), *sourceBytes))
					}
				}
			}
			for _, mod := range modifiers {
				if mod == "abstract" {
					classExtra.IsAbstract = true
				}
				if mod == "final" {
					classExtra.IsFinal = true
				}
			}
		} else if node.Kind() == "interface_declaration" {
			if superInterfacesNode := node.ChildByFieldName("superinterfaces"); superInterfacesNode != nil {
				if typeListNode := superInterfacesNode.NamedChild(0); typeListNode != nil {
					for i := 0; i < int(typeListNode.NamedChildCount()); i++ {
						classExtra.ImplementedInterfaces = append(classExtra.ImplementedInterfaces, getNodeContent(typeListNode.NamedChild(uint(i)), *sourceBytes))
					}
				}
			}
		}
		extra.ClassExtra = classExtra
	case model.Method:
		elem.Signature = extractMethodSignature(node, *sourceBytes)
		methodExtra := &model.MethodExtra{IsConstructor: node.Kind() == "constructor_declaration"}
		if node.Kind() == "method_declaration" {
			if typeNode := node.ChildByFieldName("type"); typeNode != nil {
				extra.ReturnType = getNodeContent(typeNode, *sourceBytes)
			}
		}
		if throwsNode := node.ChildByFieldName("throws"); throwsNode != nil {
			if typeListNode := throwsNode.NamedChild(0); typeListNode != nil {
				for i := 0; i < int(typeListNode.NamedChildCount()); i++ {
					methodExtra.ThrowsTypes = append(methodExtra.ThrowsTypes, getNodeContent(typeListNode.NamedChild(uint(i)), *sourceBytes))
				}
			}
		}
		if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
			if formalParamsNode := paramsNode.NamedChild(0); formalParamsNode != nil {
				for i := 0; i < int(formalParamsNode.NamedChildCount()); i++ {
					methodExtra.Parameters = append(methodExtra.Parameters, getNodeContent(formalParamsNode.NamedChild(uint(i)), *sourceBytes))
				}
			}
		}
		extra.MethodExtra = methodExtra
	case model.Field:
		fieldExtra := &model.FieldExtra{}
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			extra.Type = getNodeContent(typeNode, *sourceBytes)
		}
		for _, mod := range modifiers {
			if mod == "final" {
				fieldExtra.IsConstant = true
				break
			}
		}
		extra.FieldExtra = fieldExtra
	case model.EnumConstant:
		elem.Signature = name
		if argsNode := node.ChildByFieldName("arguments"); argsNode != nil {
			elem.Signature += getNodeContent(argsNode, *sourceBytes)
		}
	}

	if extra.MethodExtra != nil || extra.ClassExtra != nil || extra.FieldExtra != nil || len(extra.Modifiers) > 0 || extra.ReturnType != "" || extra.Type != "" {
		elem.Extra = extra
	}

	return elem, kind
}

func getNodeContent(n *sitter.Node, sourceBytes []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(sourceBytes)
}

func findNamedChildOfType(n *sitter.Node, nodeType string) *sitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(uint(i))
		if child != nil && child.Kind() == nodeType {
			return child
		}
	}
	return nil
}
