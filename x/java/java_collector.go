package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens/core"
	"github.com/CodMac/arch-lens/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Collector struct {
	resolver core.SymbolResolver
}

func NewJavaCollector() *Collector {
	return &Collector{resolver: NewJavaSymbolResolver()}
}

// =============================================================================
// 1. 核心生命周期 (Core Workflow)
// =============================================================================

func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*core.FileContext, error) {
	fCtx := core.NewFileContext(filePath, rootNode, sourceBytes)

	// Step 1: 基础声明 (Package & Imports)
	c.processTopLevelDeclarations(fCtx)

	// Step 2: 递归收集定义 (Building Tree)
	nameOccurrence := make(map[string]int)
	c.collectBasicDefinitions(fCtx.RootNode, fCtx, fCtx.PackageName, nameOccurrence)

	// Step 3: 修正特殊作用域变量的 QN
	c.refineVariableScopes(fCtx)

	// Step 4: 元数据增强 (Metadata & Signatures)
	c.enrichMetadata(fCtx)

	// Step 5: 语法糖处理 (Records, Enums, Constructors)
	c.applySyntacticSugar(fCtx)

	return fCtx, nil
}

func (c *Collector) processTopLevelDeclarations(fCtx *core.FileContext) {
	for i := 0; i < int(fCtx.RootNode.ChildCount()); i++ {
		child := fCtx.RootNode.Child(uint(i))
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "package_declaration":
			if ident := c.findNamedChildOfType(child, "scoped_identifier"); ident != nil {
				fCtx.PackageName = c.getNodeContent(ident, *fCtx.SourceBytes)
			} else if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				fCtx.PackageName = c.getNodeContent(nameNode, *fCtx.SourceBytes)
			}
		case "import_declaration":
			c.handleImport(child, fCtx)
		}
	}
}

func (c *Collector) collectBasicDefinitions(node *sitter.Node, fCtx *core.FileContext, currentQN string, occurrences map[string]int) {
	if node.IsNamed() {
		if elems, kind := c.identifyElements(node, fCtx, currentQN); len(elems) > 0 {
			for _, elem := range elems {
				c.applyUniqueQN(elem, node, currentQN, occurrences, fCtx.SourceBytes)
				fCtx.AddDefinition(elem, currentQN, node)
			}

			// 如果是作用域容器（如类或方法），继续深入
			if c.isScopeContainer(kind, node) {
				childOccurrences := make(map[string]int)
				for i := 0; i < int(node.ChildCount()); i++ {
					c.collectBasicDefinitions(node.Child(uint(i)), fCtx, elems[0].QualifiedName, childOccurrences)
				}
				return
			}
		}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		c.collectBasicDefinitions(node.Child(uint(i)), fCtx, currentQN, occurrences)
	}
}

func (c *Collector) refineVariableScopes(fCtx *core.FileContext) {
	// 获取所有已注册的 block，用于后续比对
	blocks, _ := fCtx.FindByShortName("block")
	if len(blocks) == 0 {
		return
	}

	for _, entry := range fCtx.Definitions {
		// 仅针对变量进行作用域修正
		if entry.Element.Kind != model.Variable {
			continue
		}

		// 1. 向上寻找最近的逻辑容器 (try/for/if/catch)
		containerNode := c.findNearestBlockParent(entry.Node)
		if containerNode == nil {
			continue
		}

		// 2. 遍历容器的子节点，寻找该变量逻辑上所属的 block 节点
		for i := 0; i < int(containerNode.ChildCount()); i++ {
			child := containerNode.Child(uint(i))
			// 只有当子节点是 block，且不是变量自身的定义节点时才处理
			if child.Kind() != "block" {
				continue
			}

			// 3. 在已采集的定义中，通过 Location 匹配找到对应的 block 实体
			for _, bDef := range blocks {
				if c.matchLocation(child, bDef.Element) {
					newParentQN := bDef.Element.QualifiedName

					// 4. 更新 ParentQN 并重新构建 QualifiedName
					entry.ParentQN = newParentQN
					entry.Element.QualifiedName = c.resolver.BuildQualifiedName(newParentQN, entry.Element.Name)

					// 一旦找到匹配的 block 并完成重定位，即可跳出当前变量的查找
					goto nextVariable
				}
			}
		}
	nextVariable:
	}
}

// =============================================================================
// 2. 元素识别逻辑 (Element Identification)
// =============================================================================

func (c *Collector) identifyElements(node *sitter.Node, fCtx *core.FileContext, parentQN string) ([]*model.CodeElement, model.ElementKind) {
	var kind model.ElementKind
	var names []string
	kindStr := node.Kind()

	switch kindStr {
	case "class_declaration", "record_declaration":
		kind = model.Class
	case "interface_declaration":
		kind = model.Interface
	case "enum_declaration":
		kind = model.Enum
	case "enum_constant":
		kind = model.EnumConstant
	case "annotation_type_declaration":
		kind = model.KAnnotation
	case "annotation_type_element_declaration", "method_declaration", "constructor_declaration":
		kind = model.Method
	case "field_declaration":
		kind = model.Field
		names = c.extractAllVariableNames(node, fCtx.SourceBytes)
	case "local_variable_declaration", "formal_parameter", "spread_parameter", "resource", "catch_formal_parameter":
		kind = model.Variable
		names = c.extractAllVariableNames(node, fCtx.SourceBytes)
	case "enhanced_for_statement", "instanceof_expression":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			kind = model.Variable
			names = []string{c.getNodeContent(nameNode, *fCtx.SourceBytes)}
		}
	case "lambda_expression":
		kind = model.Lambda
		names = []string{"lambda"}
	case "method_reference":
		kind = model.MethodRef
		names = []string{"method_ref"}
	case "static_initializer":
		kind = model.ScopeBlock
		names = []string{"$static"}
	case "identifier":
		if k, n := c.identifyLambdaParameter(node, fCtx); k != "" {
			kind = k
			names = []string{n}
		}
	case "block":
		kind, names = c.identifyBlockType(node)
	case "object_creation_expression":
		if c.findNamedChildOfType(node, "class_body") != nil {
			kind = model.AnonymousClass
			names = []string{"anonymousClass"}
		}
	}

	if kind != "" && names == nil {
		names = []string{c.resolveMissingName(node, kind, parentQN, fCtx.SourceBytes)}
	}
	if kind == "" || names == nil {
		return nil, ""
	}

	var elements []*model.CodeElement
	for _, name := range names {
		elements = append(elements, &model.CodeElement{
			Kind:         kind,
			Name:         name,
			Path:         fCtx.FilePath,
			Location:     c.extractLocation(node, fCtx.FilePath),
			IsFormSource: true,
		})
	}
	return elements, kind
}

func (c *Collector) identifyLambdaParameter(node *sitter.Node, fCtx *core.FileContext) (model.ElementKind, string) {
	parent := node.Parent()
	if parent == nil {
		return "", ""
	}

	pKind := parent.Kind()
	if pKind == "inferred_parameters" || pKind == "lambda_expression" {
		// 如果是单参数 Lambda (s -> ...)，确保 identifier 是参数位置而非 Body 位置
		if pKind == "lambda_expression" {
			firstChild := parent.NamedChild(0)
			if firstChild == nil || c.getNodeContent(firstChild, *fCtx.SourceBytes) != c.getNodeContent(node, *fCtx.SourceBytes) {
				return "", ""
			}
		}

		return model.Variable, c.getNodeContent(node, *fCtx.SourceBytes)
	}
	return "", ""
}

func (c *Collector) identifyBlockType(node *sitter.Node) (model.ElementKind, []string) {
	parent := node.Parent()
	if parent == nil {
		return "", nil
	}

	pKind := parent.Kind()
	if pKind == "class_body" {
		return model.ScopeBlock, []string{"$instance"}
	}

	// 排除已经拥有作用域名称的块，防止 QN 冗余
	if pKind == "method_declaration" ||
		pKind == "constructor_declaration" ||
		pKind == "static_initializer" ||
		pKind == "lambda_expression" ||
		pKind == "method_reference" {
		return "", nil
	}

	return model.ScopeBlock, []string{"block"}
}

// =============================================================================
// 3. 元数据填充 (Metadata Enrichment)
// =============================================================================

func (c *Collector) enrichMetadata(fCtx *core.FileContext) {
	for _, entry := range fCtx.Definitions {
		c.processMetadataForEntry(entry, fCtx)
	}
}

func (c *Collector) processMetadataForEntry(entry *core.DefinitionEntry, fCtx *core.FileContext) {
	node, elem := entry.Node, entry.Element
	mods, annos := c.extractModifiersAndAnnotations(node, *fCtx.SourceBytes)
	elem.Doc, elem.Comment = c.extractComments(node, fCtx.SourceBytes)

	extra := &model.Extra{Modifiers: mods, Annotations: annos, Mores: make(map[string]interface{})}
	isStatic, isFinal := c.contains(mods, "static"), c.contains(mods, "final")

	switch elem.Kind {
	case model.Method:
		if node.Kind() == "annotation_type_element_declaration" {
			c.fillAnnotationMember(elem, node, extra, fCtx)
		} else {
			c.fillMethodMetadata(elem, node, extra, mods, fCtx)
		}
	case model.Class, model.Interface, model.KAnnotation:
		c.fillTypeMetadata(elem, node, extra, mods, isFinal, fCtx)
	case model.Field:
		c.fillFieldMetadata(elem, node, extra, mods, isStatic, isFinal, fCtx)
	case model.Variable:
		c.fillLocalVariableMetadata(elem, node, extra, mods, isFinal, fCtx)
	case model.EnumConstant:
		c.fillEnumConstantMetadata(elem, node, extra, fCtx)
	case model.Lambda:
		c.fillLambdaMetadata(elem, node, extra, fCtx)
	case model.MethodRef:
		// 1. 设置原始签名 (如 System.out::println)
		elem.Signature = c.getNodeContent(node, *fCtx.SourceBytes)

		// 2. 深度解析被引用的目标
		c.fillMethodReferenceDetails(elem, node, extra, fCtx)
	case model.ScopeBlock:
		c.fillScopeBlockMetadata(elem, node, extra)
	case model.AnonymousClass:
		c.fillAnonymousClassMetadata(elem, node, extra, fCtx)
	}
	elem.Extra = extra
}

// --- Metadata Fillers ---

func (c *Collector) fillTypeMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isFinal bool, fCtx *core.FileContext) {
	extra.Mores[ClassIsAbstract], extra.Mores[ClassIsFinal] = c.contains(mods, "abstract"), isFinal
	extra.Mores[ClassIsStatic] = c.contains(mods, "static")

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = c.getNodeContent(tpNode, *fCtx.SourceBytes)
	}

	heritage := ""
	if super := node.ChildByFieldName("superclass"); super != nil {
		content := c.getNodeContent(super, *fCtx.SourceBytes)
		extra.Mores[ClassSuperClass] = strings.TrimSpace(strings.TrimPrefix(content, "extends"))
		heritage += " " + content
	}

	ifacesNode := c.findInterfacesNode(node)
	if ifacesNode != nil {
		if ifaces := c.extractInterfaceListFromNode(ifacesNode, fCtx.SourceBytes); len(ifaces) > 0 {
			mKey := InterfaceImplementedInterfaces
			if elem.Kind == model.Class {
				mKey = ClassImplementedInterfaces
			}
			extra.Mores[mKey] = ifaces
			heritage += " " + c.getNodeContent(ifacesNode, *fCtx.SourceBytes)
		}
	}

	displayKind := strings.Replace(node.Kind(), "_declaration", "", 1)
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s%s%s",
		strings.Join(mods, " "), displayKind, elem.Name, typeParams, heritage))
}

func (c *Collector) fillMethodMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, fCtx *core.FileContext) {
	extra.Mores[MethodIsConstructor] = (node.Kind() == "constructor_declaration")

	typeParams := ""
	if tpNode := node.ChildByFieldName("type_parameters"); tpNode != nil {
		typeParams = c.getNodeContent(tpNode, *fCtx.SourceBytes) + " "
	}

	retType := ""
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		retType = c.getNodeContent(tNode, *fCtx.SourceBytes)
		extra.Mores[MethodReturnType] = retType
	}

	paramsRaw := c.extractParameterWithNames(node, fCtx.SourceBytes)
	if params := c.extractParameterList(node, fCtx.SourceBytes); len(params) > 0 {
		extra.Mores[MethodParameters] = params
	}

	throwsList := c.extractThrows(node, fCtx.SourceBytes)
	throwsStr := ""
	if len(throwsList) > 0 {
		extra.Mores[MethodThrowsTypes] = throwsList
		throwsStr = " throws " + strings.Join(throwsList, ", ")
	}

	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s%s %s%s%s",
		strings.Join(mods, " "), typeParams, retType, elem.Name, paramsRaw, throwsStr))
}

func (c *Collector) fillFieldMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isStatic, isFinal bool, fCtx *core.FileContext) {
	vType := c.extractTypeString(node, fCtx.SourceBytes)
	extra.Mores[FieldRawType], extra.Mores[FieldIsStatic], extra.Mores[FieldIsFinal] = vType, isStatic, isFinal
	extra.Mores[FieldIsConstant] = isStatic && isFinal
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *Collector) fillLocalVariableMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isFinal bool, fCtx *core.FileContext) {
	vType := c.extractTypeString(node, fCtx.SourceBytes)
	extra.Mores[VariableRawType], extra.Mores[VariableIsFinal] = vType, isFinal
	extra.Mores[VariableIsParam] = (node.Kind() == "formal_parameter" || node.Kind() == "spread_parameter")
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *Collector) fillEnumConstantMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
	elem.Signature = elem.Name

	if argList := c.findNamedChildOfType(node, "argument_list"); argList != nil {
		var args []string
		for i := 0; i < int(argList.NamedChildCount()); i++ {
			args = append(args, c.getNodeContent(argList.NamedChild(uint(i)), *fCtx.SourceBytes))
		}
		extra.Mores[EnumArguments] = args
	}
}

func (c *Collector) fillAnnotationMember(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
	extra.Mores[MethodIsAnnotation] = true
	if valNode := node.ChildByFieldName("value"); valNode != nil {
		extra.Mores[MethodDefaultValue] = c.getNodeContent(valNode, *fCtx.SourceBytes)
	}
	vType := c.getNodeContent(node.ChildByFieldName("type"), *fCtx.SourceBytes)
	elem.Signature = fmt.Sprintf("%s %s()", vType, elem.Name)
}

func (c *Collector) fillScopeBlockMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra) {
	isStatic := (node.Kind() == "static_initializer")
	extra.Mores[BlockIsStatic] = isStatic
	elem.Signature = "{...}"
	if isStatic {
		elem.Signature = "static {...}"
	}
}

func (c *Collector) fillMethodReferenceDetails(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
	var receiver, target string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		kind := child.Kind()

		// 1. 忽略不需要的符号和中间件
		if kind == "::" || kind == "type_arguments" {
			if kind == "type_arguments" {
				extra.Mores[MethodRefTypeArgs] = c.getNodeContent(child, *fCtx.SourceBytes)
			}
			continue
		}

		// 2. 识别内容
		content := c.getNodeContent(child, *fCtx.SourceBytes)
		if content == "" {
			continue
		}

		// 逻辑：第一个非符号/非泛型节点是 Receiver，第二个是 Target
		if receiver == "" {
			receiver = content
		} else if target == "" {
			// 如果遇到了 new，说明是构造函数引用
			if kind == "new" {
				target = "new"
			} else {
				target = content
			}
		}
	}

	if receiver != "" {
		extra.Mores[MethodRefReceiver] = receiver
	}
	if target != "" {
		extra.Mores[MethodRefTarget] = target
	}
}

func (c *Collector) fillLambdaMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
	// 1. 提取参数部分
	// Lambda 参数可能是: (a, b) -> ... 或 a -> ... 或 (int a) -> ...
	var paramsStr string
	paramNode := node.ChildByFieldName("parameters")
	if paramNode != nil {
		paramsStr = c.getNodeContent(paramNode, *fCtx.SourceBytes)
	} else {
		// 处理单参数没有括号的情况: s -> s.toLowerCase()
		// 在 tree-sitter-java 中，这种 identifier 会是 lambda_expression 的第一个命名子节点
		if firstChild := node.NamedChild(0); firstChild != nil && firstChild.Kind() == "identifier" {
			paramsStr = c.getNodeContent(firstChild, *fCtx.SourceBytes)
		}
	}
	extra.Mores[LambdaParameters] = paramsStr

	// 2. 识别 Body 类型
	// body 可能是 block 或 表达式
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		isBlock := bodyNode.Kind() == "block"
		extra.Mores[LambdaBodyIsBlock] = isBlock

		// 生成更具描述性的 Signature，例如 (s) -> { ... } 或 (a, b) -> expr
		bodyType := "expr"
		if isBlock {
			bodyType = "{...}"
		}
		elem.Signature = fmt.Sprintf("%s -> %s", paramsStr, bodyType)
	}
}

func (c *Collector) fillAnonymousClassMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
	// 在 identifyElement 中, AnonymousClass 锚定的 node 是 "object_creation_expression"
	if node.Kind() != "object_creation_expression" {
		return
	}

	// 提取 new 关键字后的类型，例如 new Runnable() { ... } 中的 Runnable
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		typeName := c.getNodeContent(typeNode, *fCtx.SourceBytes)
		extra.Mores[AnonymousClassType] = typeName
		elem.Signature = "anonymous extends/implements " + typeName
	}
}

// =============================================================================
// 4. 语法糖处理 (Syntactic Sugar)
// =============================================================================

func (c *Collector) applySyntacticSugar(fCtx *core.FileContext) {
	clazz, ok := fCtx.FindByElementKind(model.Class)
	if ok {
		for _, entry := range clazz {
			elem, node := entry.Element, entry.Node
			if node.Kind() == "record_declaration" {
				c.desugarRecordMembers(elem, node, fCtx)
			} else if node.Kind() == "class_declaration" {
				c.desugarDefaultConstructor(elem, node, fCtx)
			}
		}
	}

	enums, ok := fCtx.FindByElementKind(model.Enum)
	if ok {
		for _, entry := range enums {
			c.desugarEnumMethods(entry.Element, entry.Node, fCtx)
		}
	}
}

func (c *Collector) desugarDefaultConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		if body.NamedChild(uint(i)).Kind() == "constructor_declaration" {
			return
		}
	}

	consName := elem.Name
	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, consName+"()")
	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          consName,
		QualifiedName: consQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public %s()", consName),
		Extra: &model.Extra{
			Mores: map[string]interface{}{MethodIsConstructor: true, MethodIsImplicit: true},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

func (c *Collector) desugarEnumMethods(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// values()
	vQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "values()")
	fCtx.AddDefinition(&model.CodeElement{
		Kind: model.Method, Name: "values", QualifiedName: vQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
		Signature: fmt.Sprintf("public static %s[] values()", elem.Name),
		Extra:     &model.Extra{Mores: map[string]interface{}{MethodIsImplicit: true}},
	}, elem.QualifiedName, node)

	// valueOf(String)
	voQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "valueOf(String)")
	fCtx.AddDefinition(&model.CodeElement{
		Kind: model.Method, Name: "valueOf", QualifiedName: voQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
		Signature: fmt.Sprintf("public static %s valueOf(String name)", elem.Name),
		Extra:     &model.Extra{Mores: map[string]interface{}{MethodIsImplicit: true}},
	}, elem.QualifiedName, node)
}

func (c *Collector) desugarRecordMembers(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	paramList := c.findNamedChildOfType(node, "formal_parameters")
	if paramList == nil {
		return
	}

	type component struct{ name, vType string }
	var comps []component
	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		child := paramList.NamedChild(uint(i))
		if child.Kind() == "formal_parameter" {
			comps = append(comps, component{
				name:  c.getNodeContent(child.ChildByFieldName("name"), *fCtx.SourceBytes),
				vType: c.getNodeContent(child.ChildByFieldName("type"), *fCtx.SourceBytes),
			})
		}
	}

	for _, comp := range comps {
		// Update Fields
		fieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, comp.name)
		if defs, _ := fCtx.FindByShortName(comp.name); len(defs) > 0 {
			for _, d := range defs {
				if d.Element.QualifiedName == fieldQN {
					d.Element.Kind = model.Field
					d.Element.Extra.Mores[FieldIsRecordComponent] = true
					d.Element.Extra.Mores[FieldIsFinal] = true
				}
			}
		}
		// Accessors
		mIdentity := comp.name + "()"
		mQN := c.resolver.BuildQualifiedName(elem.QualifiedName, mIdentity)
		if len(c.findDefinitionsByQN(fCtx, mQN)) == 0 {
			fCtx.AddDefinition(&model.CodeElement{
				Kind: model.Method, Name: comp.name, QualifiedName: mQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
				Signature: fmt.Sprintf("public %s %s()", comp.vType, comp.name),
				Extra:     &model.Extra{Mores: map[string]interface{}{MethodIsImplicit: true}},
			}, elem.QualifiedName, node)
		}
	}

	// Canonical Constructor
	var pTypes []string
	for _, comp := range comps {
		pTypes = append(pTypes, strings.TrimSpace(strings.Split(comp.vType, "<")[0]))
	}
	cIdentity := fmt.Sprintf("%s(%s)", elem.Name, strings.Join(pTypes, ","))
	cQN := c.resolver.BuildQualifiedName(elem.QualifiedName, cIdentity)
	if len(c.findDefinitionsByQN(fCtx, cQN)) == 0 {
		fCtx.AddDefinition(&model.CodeElement{
			Kind: model.Method, Name: elem.Name, QualifiedName: cQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: fmt.Sprintf("public %s(%s)", elem.Name, c.getNodeContent(paramList, *fCtx.SourceBytes)),
			Extra:     &model.Extra{Mores: map[string]interface{}{MethodIsConstructor: true, MethodIsImplicit: true}},
		}, elem.QualifiedName, node)
	}
}

// =============================================================================
// 5. 辅助工具逻辑 (Helper Utilities)
// =============================================================================

func (c *Collector) applyUniqueQN(elem *model.CodeElement, node *sitter.Node, parentQN string, occurrences map[string]int, src *[]byte) {
	identity := elem.Name
	if elem.Kind == model.Method && (node.Kind() == "method_declaration" || node.Kind() == "constructor_declaration" || node.Kind() == "annotation_type_element_declaration") {
		identity += c.extractParameterTypesOnly(node, src)
	}

	if elem.Kind == model.AnonymousClass || elem.Kind == model.Lambda || elem.Kind == model.ScopeBlock || elem.Kind == model.MethodRef {
		occurrences[elem.Name]++
		identity = fmt.Sprintf("%s$%d", elem.Name, occurrences[elem.Name])
	} else {
		occurrences[identity]++
		if occurrences[identity] > 1 {
			identity = fmt.Sprintf("%s$%d", identity, occurrences[identity])
		}
	}
	elem.QualifiedName = c.resolver.BuildQualifiedName(parentQN, identity)
}

func (c *Collector) handleImport(node *sitter.Node, fCtx *core.FileContext) {
	isStatic, isWildcard := false, false
	var pathParts []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "static":
			isStatic = true
		case "scoped_identifier", "identifier", "asterisk":
			pathParts = append(pathParts, c.getNodeContent(child, *fCtx.SourceBytes))
			if child.Kind() == "asterisk" {
				isWildcard = true
			}
		}
	}
	if len(pathParts) == 0 {
		return
	}
	fullPath := strings.Join(pathParts, ".")
	parts := strings.Split(fullPath, ".")
	alias := parts[len(parts)-1]

	entryKind := model.Class
	if isStatic {
		entryKind = model.Constant
	} else if isWildcard {
		entryKind = model.Package
	}

	fCtx.AddImport(alias, &core.ImportEntry{
		Kind: entryKind, Alias: alias, RawImportPath: fullPath, IsWildcard: isWildcard, IsStatic: isStatic, Location: c.extractLocation(node, fCtx.FilePath),
	})
}

// --- Extraction Helpers ---

func (c *Collector) extractTypeString(node *sitter.Node, src *[]byte) string {
	if node.Kind() == "identifier" {
		return "inferred"
	}
	if tNode := node.ChildByFieldName("type"); tNode != nil {
		return c.getNodeContent(tNode, *src)
	}
	if node.Kind() == "spread_parameter" {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(uint(i))
			if strings.Contains(child.Kind(), "type") {
				return c.getNodeContent(child, *src) + "..."
			}
		}
	}
	return "unknown"
}

func (c *Collector) extractParameterTypesOnly(node *sitter.Node, src *[]byte) string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return "()"
	}
	var types []string
	for i := 0; i < int(pNode.NamedChildCount()); i++ {
		tStr := strings.Split(c.extractTypeString(pNode.NamedChild(uint(i)), src), "<")[0]
		types = append(types, strings.TrimSpace(tStr))
	}
	return "(" + strings.Join(types, ",") + ")"
}

func (c *Collector) extractThrows(node *sitter.Node, src *[]byte) []string {
	tNode := c.findNamedChildOfType(node, "throws")
	if tNode == nil {
		return nil
	}
	var types []string
	for i := 0; i < int(tNode.NamedChildCount()); i++ {
		child := tNode.NamedChild(uint(i))
		if child.IsNamed() && child.Kind() != "throws" {
			types = append(types, c.getNodeContent(child, *src))
		}
	}
	return types
}

func (c *Collector) extractModifiersAndAnnotations(n *sitter.Node, src []byte) ([]string, []string) {
	var mods, annos []string
	if mNode := c.findNamedChildOfType(n, "modifiers"); mNode != nil {
		for i := 0; i < int(mNode.ChildCount()); i++ {
			child := mNode.Child(uint(i))
			txt := c.getNodeContent(child, src)
			if strings.Contains(child.Kind(), "annotation") {
				annos = append(annos, txt)
			} else if txt != "" {
				mods = append(mods, txt)
			}
		}
	}
	return mods, annos
}

func (c *Collector) extractAllVariableNames(node *sitter.Node, src *[]byte) []string {
	var names []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(uint(i))
		if child.Kind() == "variable_declarator" {
			if nNode := child.ChildByFieldName("name"); nNode != nil {
				names = append(names, c.getNodeContent(nNode, *src))
			}
		}
	}
	return names
}

func (c *Collector) extractComments(node *sitter.Node, src *[]byte) (doc, comment string) {
	curr := node
	if node.Kind() == "variable_declarator" && node.Parent() != nil {
		curr = node.Parent()
	}
	prev := curr.PrevSibling()
	for prev != nil {
		if prev.Kind() == "block_comment" || prev.Kind() == "line_comment" {
			text := c.getNodeContent(prev, *src)
			if strings.HasPrefix(text, "/**") {
				doc = text
			} else {
				comment = text
			}
			break
		}
		if strings.TrimSpace(c.getNodeContent(prev, *src)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return
}

// --- Atomic Helpers ---

func (c *Collector) isScopeContainer(k model.ElementKind, node *sitter.Node) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation,
		model.Method, model.Lambda, model.ScopeBlock, model.AnonymousClass:
		return true
	}
	return false
}

func (c *Collector) resolveMissingName(node *sitter.Node, kind model.ElementKind, parentQN string, src *[]byte) string {
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		return c.getNodeContent(nNode, *src)
	}
	if kind == model.Method {
		parts := strings.Split(parentQN, ".")
		return parts[len(parts)-1]
	}
	return ""
}

func (c *Collector) getNodeContent(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
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

func (c *Collector) extractLocation(n *sitter.Node, filePath string) *model.Location {
	return &model.Location{
		FilePath:    filePath,
		StartLine:   int(n.StartPosition().Row) + 1,
		EndLine:     int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column),
		EndColumn:   int(n.EndPosition().Column),
	}
}

func (c *Collector) matchLocation(n *sitter.Node, ele *model.CodeElement) bool {
	return (int(n.StartPosition().Row)+1 == ele.Location.StartLine) &&
		(int(n.EndPosition().Row)+1 == ele.Location.EndLine) &&
		(int(n.StartPosition().Column) == ele.Location.StartColumn) &&
		(int(n.EndPosition().Column) == ele.Location.EndColumn)
}

func (c *Collector) extractParameterList(node *sitter.Node, src *[]byte) []string {
	pNode := node.ChildByFieldName("parameters")
	if pNode == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(pNode.NamedChildCount()); i++ {
		params = append(params, c.getNodeContent(pNode.NamedChild(uint(i)), *src))
	}
	return params
}

func (c *Collector) extractParameterWithNames(node *sitter.Node, src *[]byte) string {
	if pNode := node.ChildByFieldName("parameters"); pNode != nil {
		return c.getNodeContent(pNode, *src)
	}
	return "()"
}

func (c *Collector) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (c *Collector) findInterfacesNode(node *sitter.Node) *sitter.Node {
	if n := node.ChildByFieldName("interfaces"); n != nil {
		return n
	}
	if n := node.ChildByFieldName("extends"); n != nil {
		return n
	}
	return c.findNamedChildOfType(node, "extends_interfaces")
}

func (c *Collector) extractInterfaceListFromNode(node *sitter.Node, src *[]byte) []string {
	var results []string
	target := node
	if node.Kind() != "type_list" {
		if listNode := c.findNamedChildOfType(node, "type_list"); listNode != nil {
			target = listNode
		}
	}
	for i := 0; i < int(target.NamedChildCount()); i++ {
		child := target.NamedChild(uint(i))
		if strings.Contains(child.Kind(), "type") || child.Kind() == "type_identifier" {
			results = append(results, c.getNodeContent(child, *src))
		}
	}
	return results
}

func (c *Collector) findDefinitionsByQN(fCtx *core.FileContext, qn string) []*core.DefinitionEntry {
	var result []*core.DefinitionEntry
	for _, entry := range fCtx.Definitions {
		if entry.Element.QualifiedName == qn {
			result = append(result, entry)
		}
	}
	return result
}

func (c *Collector) findNearestBlockParent(node *sitter.Node) *sitter.Node {
	// for(String s : list)
	if node.Kind() == "enhanced_for_statement" {
		return node
	}

	// 往上查找
	curr := node.Parent()
	for curr != nil {
		k := curr.Kind()
		if k == "for_statement" || k == "try_with_resources_statement" || k == "catch_clause" || k == "if_statement" {
			return curr
		}
		if k == "method_declaration" || k == "class_declaration" {
			break
		}
		curr = curr.Parent()
	}
	return nil
}
