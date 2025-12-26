package java

import (
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Collector struct{}

func NewJavaCollector() *Collector {
	return &Collector{}
}

func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*model.FileContext, error) {
	fCtx := model.NewFileContext(filePath, rootNode, sourceBytes)

	// 1. 处理顶级声明 (Package & Imports)
	c.processTopLevelDeclarations(fCtx)

	// 2. 递归收集定义
	initialQN := fCtx.PackageName
	if err := c.collectDefinitionsRecursive(fCtx.RootNode, fCtx, initialQN); err != nil {
		return nil, err
	}

	return fCtx, nil
}

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
			c.handleImport(child, fCtx)
		}
	}
}

func (c *Collector) handleImport(node *sitter.Node, fCtx *model.FileContext) {
	isStatic := false
	var pathParts []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		kind := child.Kind()
		content := c.getNodeContent(child, *fCtx.SourceBytes)

		if kind == "static" {
			isStatic = true
			continue
		}

		// 核心适配：识别所有构成导入路径的部分 (v0.23.5 会把 java.util.* 拆开)
		if kind == "scoped_identifier" || kind == "identifier" || kind == "asterisk_import" {
			pathParts = append(pathParts, content)
		} else if kind == "asterisk" {
			pathParts = append(pathParts, content)
		}
	}

	if len(pathParts) == 0 {
		return
	}

	// 拼接完整路径
	fullPath := strings.Join(pathParts, ".")
	isWildcard := strings.HasSuffix(fullPath, ".*")

	entry := &model.ImportEntry{
		RawImportPath: fullPath,
		IsWildcard:    isWildcard,
		Location:      c.extractLocation(node, fCtx.FilePath),
	}

	if isWildcard {
		entry.Alias = "*"
		entry.Kind = model.Package
	} else {
		parts := strings.Split(fullPath, ".")
		shortName := parts[len(parts)-1]
		entry.Alias = shortName
		entry.Kind = model.Class
		if isStatic {
			entry.Kind = model.Constant
		}
	}

	fCtx.AddImport(entry.Alias, entry)
}

func (c *Collector) collectDefinitionsRecursive(node *sitter.Node, fCtx *model.FileContext, currentQNPrefix string) error {
	elem, kind := c.getDefinitionElement(node, fCtx.SourceBytes, fCtx.FilePath)
	if elem != nil {
		parentQN := currentQNPrefix
		elem.QualifiedName = model.BuildQualifiedName(parentQN, elem.Name)
		fCtx.AddDefinition(elem, parentQN)

		if kind == model.Class || kind == model.Interface || kind == model.Enum || kind == model.KAnnotation || kind == model.Method {
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

func (c *Collector) getDefinitionElement(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	if node == nil {
		return nil, ""
	}

	elem, kind := c.extractElementBasic(node, sourceBytes, filePath)
	if elem == nil {
		return nil, ""
	}

	c.fillElementExtra(node, elem, kind, sourceBytes)
	return elem, kind
}

func (c *Collector) extractElementBasic(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	var kind model.ElementKind
	var nameNode *sitter.Node

	switch node.Kind() {
	case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration":
		nameNode = node.ChildByFieldName("name")
		switch node.Kind() {
		case "class_declaration":
			kind = model.Class
		case "interface_declaration":
			kind = model.Interface
		case "enum_declaration":
			kind = model.Enum
		case "annotation_type_declaration":
			kind = model.KAnnotation
		}
	case "annotation_type_element_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Method
	case "method_declaration", "constructor_declaration":
		nameNode = node.ChildByFieldName("name")
		kind = model.Method
	case "enum_constant":
		nameNode = node.ChildByFieldName("name")
		kind = model.EnumConstant
	case "field_declaration":
		if vNode := c.findNamedChildOfType(node, "variable_declarator"); vNode != nil {
			nameNode = vNode.ChildByFieldName("name")
			kind = model.Field
		}
	default:
		return nil, ""
	}

	name := ""
	if nameNode != nil {
		name = c.getNodeContent(nameNode, *sourceBytes)
	} else if node.Kind() == "constructor_declaration" {
		// 回溯获取类名作为构造函数名
		curr := node.Parent()
		for curr != nil {
			if curr.Kind() == "class_declaration" {
				if cn := curr.ChildByFieldName("name"); cn != nil {
					name = c.getNodeContent(cn, *sourceBytes)
				}
				break
			}
			curr = curr.Parent()
		}
		if name == "" {
			name = "Constructor"
		}
	}

	if name == "" {
		return nil, ""
	}

	elem := &model.CodeElement{
		Kind:     kind,
		Name:     name,
		Path:     filePath,
		Location: c.extractLocation(node, filePath),
		Doc:      c.extractComments(node, sourceBytes),
	}

	return elem, kind
}

func (c *Collector) extractComments(node *sitter.Node, sourceBytes *[]byte) string {
	var comments []string
	curr := node.PrevSibling()
	for curr != nil {
		k := curr.Kind()
		if k == "block_comment" || k == "line_comment" {
			comments = append([]string{c.getNodeContent(curr, *sourceBytes)}, comments...)
		} else if k == "modifiers" || k == "marker_annotation" || k == "annotation" {
			// 继续向上找 Javadoc
		} else {
			break
		}
		curr = curr.PrevSibling()
	}
	return strings.Join(comments, "\n")
}

func (c *Collector) fillElementExtra(node *sitter.Node, elem *model.CodeElement, kind model.ElementKind, sourceBytes *[]byte) {
	extra := &model.ElementExtra{}
	modifiers, annotations := c.extractModifiersAndAnnotations(node, *sourceBytes)
	extra.Modifiers = modifiers
	extra.Annotations = annotations

	switch kind {
	case model.Class, model.Interface, model.Enum, model.KAnnotation:
		c.fillClassExtra(node, extra, modifiers, sourceBytes)
	case model.Method:
		c.fillMethodExtra(node, extra, sourceBytes)
		elem.Signature = c.extractMethodSignature(node, *sourceBytes)
	case model.Field:
		c.fillFieldExtra(node, extra, modifiers, sourceBytes)
	}

	// 最终检查：如果没有任何有效元数据，则不挂载 Extra
	if extra.MethodExtra != nil || extra.ClassExtra != nil || extra.FieldExtra != nil ||
		len(extra.Modifiers) > 0 || len(extra.Annotations) > 0 {
		elem.Extra = extra
	}
}

func (c *Collector) extractMethodSignature(node *sitter.Node, sourceBytes []byte) string {
	var sb strings.Builder

	// 1. 获取修饰符
	mods, _ := c.extractModifiersAndAnnotations(node, sourceBytes)
	if len(mods) > 0 {
		sb.WriteString(strings.Join(mods, " "))
		sb.WriteString(" ")
	}

	// 2. 返回类型
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		sb.WriteString(c.getNodeContent(tNode, sourceBytes))
		sb.WriteString(" ")
	}

	// 3. 方法名
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		sb.WriteString(c.getNodeContent(nNode, sourceBytes))
	}

	// 4. 参数部分 (关键修复点)
	pNode := node.ChildByFieldName("parameters")
	if pNode != nil {
		sb.WriteString(c.getNodeContent(pNode, sourceBytes))
	} else {
		// 如果是方法定义或注解元素，即使没参数也要补上 ()
		kind := node.Kind()
		if kind == "method_declaration" || kind == "annotation_type_element_declaration" {
			sb.WriteString("()")
		}
	}

	return strings.TrimSpace(sb.String())
}

func (c *Collector) fillClassExtra(node *sitter.Node, extra *model.ElementExtra, modifiers []string, sourceBytes *[]byte) {
	ce := &model.ClassExtra{}
	for _, m := range modifiers {
		if m == "abstract" {
			ce.IsAbstract = true
		}
		if m == "final" {
			ce.IsFinal = true
		}
	}

	// 1. 处理类继承 (extends)
	if scNode := node.ChildByFieldName("superclass"); scNode != nil {
		ce.SuperClass = c.getNodeContent(scNode, *sourceBytes)
	}

	// 2. 处理接口列表 (implements 或接口的 extends)
	// 兼容 class 的 "interfaces" 字段 和 interface 的 "extends_interfaces" 节点
	var iNode *sitter.Node
	if n := node.ChildByFieldName("interfaces"); n != nil {
		iNode = n
	} else {
		// 手动查找 extends_interfaces (针对 interface 声明)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child.Kind() == "extends_interfaces" {
				iNode = child
				break
			}
		}
	}

	if iNode != nil {
		c.recursiveCollectTypes(iNode, &ce.ImplementedInterfaces, sourceBytes)
	}
	extra.ClassExtra = ce
}

func (c *Collector) fillMethodExtra(node *sitter.Node, extra *model.ElementExtra, sourceBytes *[]byte) {
	me := &model.MethodExtra{IsConstructor: node.Kind() == "constructor_declaration"}

	if tNode := node.ChildByFieldName("type"); tNode != nil {
		me.ReturnType = c.getNodeContent(tNode, *sourceBytes)
	}

	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		for i := 0; i < int(pNode.NamedChildCount()); i++ {
			me.Parameters = append(me.Parameters, c.getNodeContent(pNode.NamedChild(uint(i)), *sourceBytes))
		}
	}

	// 核心修复：根据 AST，throws 节点直接包含 type_identifier
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child.Kind() == "throws" {
			c.recursiveCollectTypes(child, &me.ThrowsTypes, sourceBytes)
			break
		}
	}
	extra.MethodExtra = me
}

func (c *Collector) fillFieldExtra(node *sitter.Node, extra *model.ElementExtra, modifiers []string, sourceBytes *[]byte) {
	fe := &model.FieldExtra{}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		fe.Type = c.getNodeContent(tNode, *sourceBytes)
	}
	for _, m := range modifiers {
		if m == "final" {
			fe.IsConstant = true
			break
		}
	}
	extra.FieldExtra = fe
}

func (c *Collector) extractModifiersAndAnnotations(n *sitter.Node, sourceBytes []byte) ([]string, []string) {
	var modifiers []string
	var annotations []string

	mNode := n.ChildByFieldName("modifiers")
	if mNode == nil {
		// 尝试手动查找 modifiers 节点
		for i := 0; i < int(n.ChildCount()); i++ {
			if n.Child(uint(i)).Kind() == "modifiers" {
				mNode = n.Child(uint(i))
				break
			}
		}
	}

	if mNode == nil {
		return nil, nil
	}

	for i := 0; i < int(mNode.ChildCount()); i++ {
		child := mNode.Child(uint(i))
		k := child.Kind()

		if k == "marker_annotation" || k == "annotation" {
			annotations = append(annotations, c.getNodeContent(child, sourceBytes))
		} else {
			// 关键修复：v0.23.5 的 default 可能是匿名节点，不要只判断 IsNamed()
			txt := c.getNodeContent(child, sourceBytes)
			if txt != "" && k != "marker_annotation" && k != "annotation" {
				modifiers = append(modifiers, txt)
			}
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

func (c *Collector) getNodeContent(n *sitter.Node, sourceBytes []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(sourceBytes)
}

func (c *Collector) findNamedChildOfType(n *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(uint(i))
		if child.Kind() == nodeType {
			return child
		}
	}
	return nil
}

// 辅助函数：确保能抓取到 type_identifier, generic_type 等
func (c *Collector) recursiveCollectTypes(n *sitter.Node, results *[]string, sourceBytes *[]byte) {
	kind := n.Kind()
	if kind == "type_identifier" || kind == "scoped_type_identifier" || kind == "generic_type" || kind == "void_type" {
		*results = append(*results, c.getNodeContent(n, *sourceBytes))
		return
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		c.recursiveCollectTypes(n.Child(uint(i)), results, sourceBytes)
	}
}
