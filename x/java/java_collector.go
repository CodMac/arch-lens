package java

import (
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

	// 1. 独立处理 Package Name (作为 QN 的前缀)
	c.collectPackageName(fCtx)

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

// collectPackageName 独立收集 Package Name
func (c *Collector) collectPackageName(fCtx *model.FileContext) {
	var pkgNode *sitter.Node

	// BUG FIX: package_declaration 是 program 的非命名子节点，不能使用 ChildByFieldName 查找。
	// 遍历根节点的所有直接子节点，查找 kind 为 "package_declaration" 的节点。
	for i := 0; i < int(fCtx.RootNode.ChildCount()); i++ {
		child := fCtx.RootNode.Child(uint(i))
		if child != nil && child.Kind() == "package_declaration" {
			pkgNode = child
			break
		}
	}

	if pkgNode != nil {
		// 修复：package_declaration 的命名子节点 (scoped_identifier/identifier) 没有 field name "name"。
		// 应该直接获取第一个命名子节点 (索引 0)。
		if pkgNameNode := pkgNode.NamedChild(0); pkgNameNode != nil {
			fCtx.PackageName = getNodeContent(pkgNameNode, *fCtx.SourceBytes)
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

		// 2. 注册定义
		fCtx.AddDefinition(elem, parentQN)

		// 3. 更新 QN 前缀
		// 对于 Class/Interface/Enum/Method，它们是容器，新的 QN 是其自身的 QualifiedName
		if kind == model.Class || kind == model.Interface || kind == model.Enum || kind == model.Method {
			currentQNPrefix = elem.QualifiedName
		}
	}

	// 递归遍历子节点
	cursor := node.Walk()
	defer cursor.Close()

	if cursor.GotoFirstChild() {
		for {
			// 递归调用，并传入当前 QN 前缀
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

// getDefinitionElement 辅助函数
func getDefinitionElement(node *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	if node == nil {
		return nil, ""
	}

	switch node.Kind() {
	case "class_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			return &model.CodeElement{Kind: model.Class, Name: getNodeContent(nameNode, *sourceBytes), Path: filePath}, model.Class
		}
	case "interface_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			return &model.CodeElement{Kind: model.Interface, Name: getNodeContent(nameNode, *sourceBytes), Path: filePath}, model.Interface
		}
	case "enum_declaration": // 收集枚举类型
		nameNode := node.ChildByFieldName("name")
		// 假设 model 包中定义了 model.Enum
		if nameNode != nil {
			// 修复: 统一返回类型为 model.Enum
			return &model.CodeElement{Kind: model.Enum, Name: getNodeContent(nameNode, *sourceBytes), Path: filePath}, model.Enum
		}
	case "enum_constant": // 收集枚举常量
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			return &model.CodeElement{Kind: model.EnumConstant, Name: getNodeContent(nameNode, *sourceBytes), Path: filePath}, model.EnumConstant
		}
	case "method_declaration", "constructor_declaration":
		nameNode := node.ChildByFieldName("name")
		name := ""
		if nameNode != nil {
			name = getNodeContent(nameNode, *sourceBytes)
		} else if node.Kind() == "constructor_declaration" {
			if parent := node.Parent(); parent != nil && (parent.Kind() == "class_declaration" || parent.Kind() == "enum_declaration") {
				// 构造函数名称通常与其父类/父枚举名称相同
				if classNameNode := parent.ChildByFieldName("name"); classNameNode != nil {
					name = getNodeContent(classNameNode, *sourceBytes)
				}
			}
			if name == "" {
				name = "Constructor"
			}
		}

		if name != "" {
			return &model.CodeElement{Kind: model.Method, Name: name, Path: filePath}, model.Method
		}
	case "field_declaration":
		// 字段声明可能包含多个变量声明，我们只取第一个
		if vNode := findNamedChildOfType(node, "variable_declarator"); vNode != nil {
			if nameNode := vNode.ChildByFieldName("name"); nameNode != nil {
				return &model.CodeElement{Kind: model.Field, Name: getNodeContent(nameNode, *sourceBytes), Path: filePath}, model.Field
			}
		}
	}
	return nil, ""
}

// getNodeContent 获取 AST 节点对应的源码文本内容
func getNodeContent(n *sitter.Node, sourceBytes []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(sourceBytes)
}

// findNamedChildOfType 查找特定类型的命名子节点
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
