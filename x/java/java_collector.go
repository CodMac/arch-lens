package java

import (
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Collector 实现了 collector.Collector 接口，用于扫描 Java 源码并提取所有定义信息（类、方法、字段等）
type Collector struct{}

func NewJavaCollector() *Collector {
	return &Collector{}
}

// CollectDefinitions 扫描抽象语法树，提取包名、导入列表以及所有代码实体的定义
func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*model.FileContext, error) {
	fCtx := model.NewFileContext(filePath, rootNode, sourceBytes)

	// 1. 处理顶级声明：Package Name 和 Import Declarations
	c.processTopLevelDeclarations(fCtx)

	// 2. 递归收集定义，维护全限定名 (QN) 栈
	initialQN := ""
	if fCtx.PackageName != "" {
		initialQN = fCtx.PackageName
	}

	if err := c.collectDefinitionsRecursive(fCtx.RootNode, fCtx, initialQN); err != nil {
		return nil, err
	}

	return fCtx, nil
}

// processTopLevelDeclarations 提取包定义和导入列表
func (c *Collector) processTopLevelDeclarations(fCtx *model.FileContext) {
	for i := 0; i < int(fCtx.RootNode.ChildCount()); i++ {
		child := fCtx.RootNode.Child(uint(i))
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "package_declaration":
			if pkgNameNode := child.NamedChild(0); pkgNameNode != nil {
				fCtx.PackageName = c.getNodeContent(pkgNameNode, *fCtx.SourceBytes)
			}
		case "import_declaration":
			if importNameNode := child.NamedChild(0); importNameNode != nil {
				fullImportPath := c.getNodeContent(importNameNode, *fCtx.SourceBytes)
				parts := strings.Split(fullImportPath, ".")
				if len(parts) > 0 {
					shortName := parts[len(parts)-1]
					fCtx.Imports[shortName] = fullImportPath
				}
			}
		}
	}
}

// collectDefinitionsRecursive 递归遍历 AST，根据层级关系构建 Qualified Name 并注册到 FileContext
func (c *Collector) collectDefinitionsRecursive(node *sitter.Node, fCtx *model.FileContext, currentQNPrefix string) error {
	if elem, kind := c.getDefinitionElement(node, fCtx.SourceBytes, fCtx.FilePath); elem != nil {
		// 1. 构造并设置 Qualified Name
		parentQN := currentQNPrefix
		elem.QualifiedName = model.BuildQualifiedName(parentQN, elem.Name)

		// 2. 注册定义到当前文件上下文
		fCtx.AddDefinition(elem, parentQN)

		// 3. 更新 QN 前缀（如果是容器类节点，后续子节点将以此为父前缀）
		if kind == model.Class || kind == model.Interface || kind == model.Enum || kind == model.Method {
			currentQNPrefix = elem.QualifiedName
		}
	}

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

// getDefinitionElement 识别节点类型，提取基础信息及元数据 (Extra)
func (c *Collector) getDefinitionElement(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	if node == nil {
		return nil, ""
	}

	// 1. 提取基础信息：Name, Kind, Location
	elem, kind := c.extractElementBasic(node, sourceBytes, filePath)
	if elem == nil {
		return nil, ""
	}

	// 2. 提取扩展信息：Modifiers, Annotations, Signatures, etc.
	c.fillElementExtra(node, elem, kind, sourceBytes)

	return elem, kind
}

// extractElementBasic 识别节点是否为定义，并返回初步构造的 CodeElement
func (c *Collector) extractElementBasic(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	var kind model.ElementKind
	var nameNode *sitter.Node

	switch node.Kind() {
	case "class_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Class
	case "interface_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Interface
	case "annotation_type_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.KAnnotation
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
		if vNode := c.findNamedChildOfType(node, "variable_declarator"); vNode != nil {
			nameNode = vNode.ChildByFieldName("name")
			kind = model.Field
		}
	default:
		return nil, ""
	}

	// 特殊处理：构造函数如果没有显式 nameNode，则向上查找类名
	name := ""
	if nameNode != nil {
		name = c.getNodeContent(nameNode, *sourceBytes)
	} else if node.Kind() == "constructor_declaration" {
		if parent := node.Parent(); parent != nil {
			if pNameNode := parent.ChildByFieldName("name"); pNameNode != nil {
				name = c.getNodeContent(pNameNode, *sourceBytes)
			}
		}
		if name == "" {
			name = "Constructor"
		}
	}

	if name == "" {
		return nil, ""
	}

	return &model.CodeElement{
		Kind:     kind,
		Name:     name,
		Path:     filePath,
		Location: c.extractLocation(node, filePath),
	}, kind
}

// fillElementExtra 填充 CodeElement 的 Extra 扩展字段
func (c *Collector) fillElementExtra(node *sitter.Node, elem *model.CodeElement, kind model.ElementKind, sourceBytes *[]byte) {
	extra := &model.ElementExtra{}
	modifiers, annotations := c.extractModifiersAndAnnotations(node, *sourceBytes)
	extra.Modifiers = modifiers
	extra.Annotations = annotations

	switch kind {
	case model.Class, model.Interface, model.Enum:
		c.fillClassExtra(node, extra, modifiers, sourceBytes)
	case model.Method:
		elem.Signature = c.extractMethodSignature(node, *sourceBytes)
		c.fillMethodExtra(node, extra, sourceBytes)
	case model.Field:
		c.fillFieldExtra(node, extra, modifiers, sourceBytes)
	case model.EnumConstant:
		elem.Signature = elem.Name
		if argsNode := node.ChildByFieldName("arguments"); argsNode != nil {
			elem.Signature += c.getNodeContent(argsNode, *sourceBytes)
		}
	}

	// 只有当有内容时才挂载 Extra 对象
	if extra.MethodExtra != nil || extra.ClassExtra != nil || extra.FieldExtra != nil ||
		len(extra.Modifiers) > 0 || extra.ReturnType != "" || extra.Type != "" {
		elem.Extra = extra
	}
}

func (c *Collector) fillClassExtra(node *sitter.Node, extra *model.ElementExtra, modifiers []string, sourceBytes *[]byte) {
	ce := &model.ClassExtra{}
	if node.Kind() == "class_declaration" {
		if ext := node.ChildByFieldName("superclass"); ext != nil {
			if typeNode := ext.NamedChild(0); typeNode != nil {
				ce.SuperClass = c.getNodeContent(typeNode, *sourceBytes)
			}
		}
		for _, mod := range modifiers {
			if mod == "abstract" {
				ce.IsAbstract = true
			}
			if mod == "final" {
				ce.IsFinal = true
			}
		}
	}
	// 提取接口实现
	if intfs := node.ChildByFieldName("interfaces"); intfs != nil {
		if list := intfs.NamedChild(0); list != nil {
			for i := 0; i < int(list.NamedChildCount()); i++ {
				ce.ImplementedInterfaces = append(ce.ImplementedInterfaces, c.getNodeContent(list.NamedChild(uint(i)), *sourceBytes))
			}
		}
	} else if node.Kind() == "interface_declaration" {
		// 接口的继承在 Java Tree-sitter 中可能对应 superinterfaces
		if sIntfs := node.ChildByFieldName("superinterfaces"); sIntfs != nil {
			if list := sIntfs.NamedChild(0); list != nil {
				for i := 0; i < int(list.NamedChildCount()); i++ {
					ce.ImplementedInterfaces = append(ce.ImplementedInterfaces, c.getNodeContent(list.NamedChild(uint(i)), *sourceBytes))
				}
			}
		}
	}
	extra.ClassExtra = ce
}

func (c *Collector) fillMethodExtra(node *sitter.Node, extra *model.ElementExtra, sourceBytes *[]byte) {
	me := &model.MethodExtra{IsConstructor: node.Kind() == "constructor_declaration"}
	if node.Kind() == "method_declaration" {
		if tNode := node.ChildByFieldName("type"); tNode != nil {
			extra.ReturnType = c.getNodeContent(tNode, *sourceBytes)
		}
	}
	// 提取参数
	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		for i := 0; i < int(pNode.NamedChildCount()); i++ {
			me.Parameters = append(me.Parameters, c.getNodeContent(pNode.NamedChild(uint(i)), *sourceBytes))
		}
	}
	// 提取异常抛出
	if tNode := node.ChildByFieldName("throws"); tNode != nil {
		if list := tNode.NamedChild(0); list != nil {
			for i := 0; i < int(list.NamedChildCount()); i++ {
				me.ThrowsTypes = append(me.ThrowsTypes, c.getNodeContent(list.NamedChild(uint(i)), *sourceBytes))
			}
		}
	}
	extra.MethodExtra = me
}

func (c *Collector) fillFieldExtra(node *sitter.Node, extra *model.ElementExtra, modifiers []string, sourceBytes *[]byte) {
	fe := &model.FieldExtra{}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		extra.Type = c.getNodeContent(tNode, *sourceBytes)
	}
	for _, mod := range modifiers {
		if mod == "final" {
			fe.IsConstant = true
			break
		}
	}
	extra.FieldExtra = fe
}

// --- 辅助工具方法 ---

func (c *Collector) extractLocation(n *sitter.Node, filePath string) *model.Location {
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

func (c *Collector) extractModifiersAndAnnotations(n *sitter.Node, sourceBytes []byte) ([]string, []string) {
	var modifiers []string
	var annotations []string
	var modifiersNode *sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		if child := n.Child(uint(i)); child.Kind() == "modifiers" {
			modifiersNode = child
			break
		}
	}
	if modifiersNode == nil {
		return modifiers, annotations
	}

	content := c.getNodeContent(modifiersNode, sourceBytes)
	for _, part := range strings.Fields(content) {
		if c.isKeywordModifier(part) {
			modifiers = append(modifiers, part)
		}
	}
	for i := 0; i < int(modifiersNode.ChildCount()); i++ {
		child := modifiersNode.Child(uint(i))
		if strings.Contains(child.Kind(), "annotation") {
			annotations = append(annotations, c.getNodeContent(child, sourceBytes))
		}
	}
	return modifiers, annotations
}

func (c *Collector) isKeywordModifier(s string) bool {
	switch s {
	case "public", "private", "protected", "static", "final", "abstract",
		"synchronized", "transient", "volatile", "default", "native", "strictfp":
		return true
	}
	return false
}

func (c *Collector) extractMethodSignature(node *sitter.Node, sourceBytes []byte) string {
	var parts []string
	mods, _ := c.extractModifiersAndAnnotations(node, sourceBytes)
	if len(mods) > 0 {
		parts = append(parts, strings.Join(mods, " "))
	}

	if node.Kind() == "method_declaration" {
		if tNode := node.ChildByFieldName("type"); tNode != nil {
			parts = append(parts, c.getNodeContent(tNode, sourceBytes))
		}
	}

	name := ""
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		name = c.getNodeContent(nNode, sourceBytes)
	} else if node.Kind() == "constructor_declaration" {
		if p := node.Parent(); p != nil {
			if cnNode := p.ChildByFieldName("name"); cnNode != nil {
				name = c.getNodeContent(cnNode, sourceBytes)
			}
		}
	}
	if name != "" {
		parts = append(parts, name)
	}

	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		parts = append(parts, c.getNodeContent(pNode, sourceBytes))
	} else {
		parts = append(parts, "()")
	}
	return strings.Join(parts, " ")
}

func (c *Collector) getNodeContent(n *sitter.Node, sourceBytes []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(sourceBytes)
}

func (c *Collector) findNamedChildOfType(n *sitter.Node, nodeType string) *sitter.Node {
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
