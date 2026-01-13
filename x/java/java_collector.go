package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Collector struct {
	resolver core.SymbolResolver
}

func NewJavaCollector() *Collector {
	resolver, err := core.GetSymbolResolver(core.LangJava)
	if err != nil {
		panic(err)
	}
	return &Collector{resolver: resolver}
}

// ==========================================
// 1. 核心生命周期 (Core Workflow)
// ==========================================

func (c *Collector) CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*core.FileContext, error) {
	fCtx := core.NewFileContext(filePath, rootNode, sourceBytes)

	// 1. 处理包声明和导入声明
	c.processTopLevelDeclarations(fCtx)

	// 2. 递归收集基础定义
	nameOccurrence := make(map[string]int)
	c.collectBasicDefinitions(fCtx.RootNode, fCtx, fCtx.PackageName, nameOccurrence)

	// 3. 二次遍历填充元数据
	c.enrichMetadata(fCtx)

	// 4. 语法糖增强 (Desugaring)
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
		if elem, kind := c.identifyElement(node, fCtx, currentQN); elem != nil {
			// 严格保持你验证过的 QN 生成逻辑
			c.applyUniqueQN(elem, node, currentQN, occurrences, fCtx.SourceBytes)
			fCtx.AddDefinition(elem, currentQN, node)

			if c.isScopeContainer(kind, node) {
				childOccurrences := make(map[string]int)
				for i := 0; i < int(node.ChildCount()); i++ {
					c.collectBasicDefinitions(node.Child(uint(i)), fCtx, elem.QualifiedName, childOccurrences)
				}
				return
			}
		}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		c.collectBasicDefinitions(node.Child(uint(i)), fCtx, currentQN, occurrences)
	}
}

func (c *Collector) applySyntacticSugar(fCtx *core.FileContext) {
	var allEntries []*core.DefinitionEntry
	for _, entries := range fCtx.DefinitionsBySN {
		allEntries = append(allEntries, entries...)
	}

	for _, entry := range allEntries {
		elem := entry.Element
		node := entry.Node

		switch elem.Kind {
		case model.Class:
			// 区分普通类和 Record
			if node.Kind() == "record_declaration" {
				c.desugarRecordMembers(elem, node, fCtx)
			} else if node.Kind() == "class_declaration" {
				// 只有真正的 class 才补全默认构造函数
				c.desugarDefaultConstructor(elem, node, fCtx)
			}
		case model.Enum:
			c.desugarEnumMethods(elem, node, fCtx)
		}
	}
}

// ==========================================
// 2. 元素识别逻辑 (Element Identification)
// ==========================================

func (c *Collector) identifyElement(node *sitter.Node, fCtx *core.FileContext, parentQN string) (*model.CodeElement, model.ElementKind) {
	var kind model.ElementKind
	var name string
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
		name = c.getNodeContent(node.ChildByFieldName("name"), *fCtx.SourceBytes)
	case "annotation_type_declaration":
		kind, name = model.KAnnotation, c.getNodeContent(node.ChildByFieldName("name"), *fCtx.SourceBytes)
	case "annotation_type_element_declaration":
		kind, name = model.Method, c.getNodeContent(node.ChildByFieldName("name"), *fCtx.SourceBytes)
	case "method_declaration", "constructor_declaration":
		kind = model.Method
	case "field_declaration", "local_variable_declaration":
		kind = c.determineVariableKind(kindStr)
		name = c.extractVariableName(node, fCtx.SourceBytes)
	case "formal_parameter", "spread_parameter":
		kind, name = model.Variable, c.extractVariableName(node, fCtx.SourceBytes)
	case "resource": // try-catch 参数
		kind = model.Variable
		name = c.getNodeContent(node.ChildByFieldName("name"), *fCtx.SourceBytes)
	case "lambda_expression":
		kind, name = model.Lambda, "lambda"
	case "static_initializer":
		kind, name = model.ScopeBlock, "$static"
	case "block":
		kind, name = c.identifyBlockType(node)
	case "object_creation_expression":
		if c.findNamedChildOfType(node, "class_body") != nil {
			kind, name = model.AnonymousClass, "anonymousClass"
		}
	}

	if kind != "" && name == "" {
		name = c.resolveMissingName(node, kind, parentQN, fCtx.SourceBytes)
	}
	if kind == "" || name == "" {
		return nil, ""
	}

	return &model.CodeElement{
		Kind:     kind,
		Name:     name,
		Path:     fCtx.FilePath,
		Location: c.extractLocation(node, fCtx.FilePath),
	}, kind
}

// ==========================================
// 3. 元数据填充 (Metadata Enrichment)
// ==========================================

func (c *Collector) enrichMetadata(fCtx *core.FileContext) {
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			c.processMetadataForEntry(entry, fCtx)
		}
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
		c.fillEnumConstantMetadata(node, extra, fCtx)
		elem.Signature = elem.Name
	case model.ScopeBlock:
		c.fillScopeBlockMetadata(elem, node, extra)
	}
	elem.Extra = extra
}

// ==========================================
// 4. 私有填充工具 (Metadata Sub-fillers)
// ==========================================

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

	// 1. 提取方法级泛型 (例如 <E extends Exception>)
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

	// 2. 组装 Signature 时加入 typeParams
	// 格式：[修饰符] [泛型] [返回类型] [方法名]([参数]) [异常]
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s%s %s%s%s",
		strings.Join(mods, " "), typeParams, retType, elem.Name, paramsRaw, throwsStr))
}

func (c *Collector) fillFieldMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isStatic, isFinal bool, fCtx *core.FileContext) {
	vType := c.extractTypeString(node, fCtx.SourceBytes)

	// 填充 Field 专属元数据
	extra.Mores[FieldType] = vType
	extra.Mores[FieldIsStatic] = isStatic
	extra.Mores[FieldIsFinal] = isFinal
	extra.Mores[FieldIsConstant] = isStatic && isFinal // 方便后续判断是否为常量引用

	// 构建 Signature: [public static] String myField
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *Collector) fillLocalVariableMetadata(elem *model.CodeElement, node *sitter.Node, extra *model.Extra, mods []string, isFinal bool, fCtx *core.FileContext) {
	vType := c.extractTypeString(node, fCtx.SourceBytes)

	// 填充 Variable 专属元数据
	extra.Mores[VariableType] = vType
	extra.Mores[VariableIsFinal] = isFinal

	// 区分是普通局部变量还是方法参数
	extra.Mores[VariableIsParam] = (node.Kind() == "formal_parameter" || node.Kind() == "spread_parameter")

	// 构建 Signature: [final] int count
	elem.Signature = strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.Join(mods, " "), vType, elem.Name))
}

func (c *Collector) fillEnumConstantMetadata(node *sitter.Node, extra *model.Extra, fCtx *core.FileContext) {
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

// ==========================================
// 5. 语法糖增强 (Sugar)
// ==========================================

func (c *Collector) desugarDefaultConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// 查找类体中是否已经有了显式的构造函数
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	hasConstructor := false
	for i := 0; i < int(body.NamedChildCount()); i++ {
		if body.NamedChild(uint(i)).Kind() == "constructor_declaration" {
			hasConstructor = true
			break
		}
	}

	if !hasConstructor {
		// 补全一个无参构造函数
		consName := elem.Name
		// 构造函数 QN 通常不带参数名，仅带类型，无参即为 ()
		consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, consName+"()")

		consElem := &model.CodeElement{
			Kind:          model.Method,
			Name:          consName,
			QualifiedName: consQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location, // 指向类声明的位置
			Signature:     fmt.Sprintf("public %s()", consName),
			Extra: &model.Extra{
				Mores: map[string]interface{}{
					MethodIsConstructor: true,
					MethodIsImplicit:    true,
				},
			},
		}
		// 注入到 FileContext
		fCtx.AddDefinition(consElem, elem.QualifiedName, node)
	}
}

func (c *Collector) desugarEnumMethods(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	// 1. 补全 values()
	valuesQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "values()")
	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          "values",
		QualifiedName: valuesQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public static %s[] values()", elem.Name),
		Extra: &model.Extra{
			Mores: map[string]interface{}{MethodIsImplicit: true},
		},
	}, elem.QualifiedName, node)

	// 2. 补全 valueOf(String)
	valueOfQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "valueOf(String)")
	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          "valueOf",
		QualifiedName: valueOfQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public static %s valueOf(String name)", elem.Name),
		Extra: &model.Extra{
			Mores: map[string]interface{}{MethodIsImplicit: true},
		},
	}, elem.QualifiedName, node)
}

func (c *Collector) desugarRecordMembers(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	paramList := c.findNamedChildOfType(node, "formal_parameters")
	if paramList == nil {
		return
	}

	// 1. 提取组件信息
	type component struct {
		name  string
		vType string
	}
	var comps []component
	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		child := paramList.NamedChild(uint(i))
		if child.Kind() == "formal_parameter" {
			name := c.getNodeContent(child.ChildByFieldName("name"), *fCtx.SourceBytes)
			vType := c.getNodeContent(child.ChildByFieldName("type"), *fCtx.SourceBytes)
			comps = append(comps, component{name, vType})
		}
	}

	// 2. 更新已有变量的元数据（Record 参数本质就是 Field），不需要 AddDefinition，因为 collectBasicDefinitions 已经加过了
	for _, comp := range comps {
		fieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, comp.name)
		if defs := fCtx.DefinitionsBySN[comp.name]; len(defs) > 0 {
			for _, d := range defs {
				if d.Element.QualifiedName == fieldQN {
					// 修改 Kind 为 Field 并添加标记
					d.Element.Kind = model.Field
					d.Element.Extra.Mores[FieldIsRecordComponent] = true
					d.Element.Extra.Mores[FieldIsFinal] = true
				}
			}
		}
	}

	// 3. 生成隐式 Accessor 方法
	for _, comp := range comps {
		methodName := comp.name
		methodIdentity := methodName + "()"
		methodQN := c.resolver.BuildQualifiedName(elem.QualifiedName, methodIdentity)

		// 重点：使用 c.findDefinitionsByQN 来精准判定是否存在
		existing := c.findDefinitionsByQN(fCtx, methodQN)
		if len(existing) == 0 {
			fCtx.AddDefinition(&model.CodeElement{
				Kind:          model.Method,
				Name:          methodName,
				QualifiedName: methodQN,
				Path:          fCtx.FilePath,
				Location:      elem.Location,
				Signature:     fmt.Sprintf("public %s %s()", comp.vType, methodName),
				Extra:         &model.Extra{Mores: map[string]interface{}{MethodIsImplicit: true}},
			}, elem.QualifiedName, node)
		}
	}

	// 4. 生成规范构造函数 (修正签名)
	var paramTypes []string
	for _, comp := range comps {
		// 这里的处理必须与 extractParameterTypesOnly 一致
		tStr := strings.Split(comp.vType, "<")[0]
		paramTypes = append(paramTypes, strings.TrimSpace(tStr))
	}
	consIdentity := fmt.Sprintf("%s(%s)", elem.Name, strings.Join(paramTypes, ","))
	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, consIdentity)

	// 检查是否已存在显式构造函数
	if len(c.findDefinitionsByQN(fCtx, consQN)) == 0 {
		fCtx.AddDefinition(&model.CodeElement{
			Kind:          model.Method,
			Name:          elem.Name,
			QualifiedName: consQN,
			Path:          fCtx.FilePath,
			Location:      elem.Location,
			Signature:     fmt.Sprintf("public %s(%s)", elem.Name, c.getNodeContent(paramList, *fCtx.SourceBytes)),
			Extra:         &model.Extra{Mores: map[string]interface{}{MethodIsConstructor: true, MethodIsImplicit: true}},
		}, elem.QualifiedName, node)
	}
}

// ==========================================
// 6. 辅助工具逻辑 (Helper Utilities)
// ==========================================

func (c *Collector) handleImport(node *sitter.Node, fCtx *core.FileContext) {
	isStatic := false
	isWildcard := false
	var pathParts []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		kind := child.Kind()
		if kind == "static" {
			isStatic = true
			continue
		}
		if kind == "scoped_identifier" || kind == "identifier" || kind == "asterisk" {
			pathParts = append(pathParts, c.getNodeContent(child, *fCtx.SourceBytes))
			if kind == "asterisk" {
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

	var entryKind model.ElementKind
	if isStatic {
		entryKind = model.Constant
	} else if isWildcard {
		entryKind = model.Package
	} else {
		entryKind = model.Class
	}

	entry := &core.ImportEntry{
		Kind:          entryKind,
		Alias:         alias,
		RawImportPath: fullPath,
		IsWildcard:    isWildcard,
		IsStatic:      isStatic,
		Location:      c.extractLocation(node, fCtx.FilePath),
	}
	fCtx.AddImport(alias, entry)
}

func (c *Collector) applyUniqueQN(elem *model.CodeElement, node *sitter.Node, parentQN string, occurrences map[string]int, src *[]byte) {
	identity := elem.Name
	// 严格遵循之前的判断条件，防止构造函数或重载方法匹配失效
	if elem.Kind == model.Method &&
		(node.Kind() == "method_declaration" || node.Kind() == "constructor_declaration" || node.Kind() == "annotation_type_element_declaration") {
		identity += c.extractParameterTypesOnly(node, src)
	}
	occurrences[identity]++
	count := occurrences[identity]
	if elem.Kind == model.AnonymousClass || elem.Kind == model.Lambda || elem.Kind == model.ScopeBlock || count > 1 {
		identity = fmt.Sprintf("%s$%d", identity, count)
	}
	elem.QualifiedName = c.resolver.BuildQualifiedName(parentQN, identity)
}

func (c *Collector) extractTypeString(node *sitter.Node, src *[]byte) string {
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
		param := pNode.NamedChild(uint(i))
		// 移除泛型部分，仅保留基本类名用于签名区分
		tStr := strings.Split(c.extractTypeString(param, src), "<")[0]
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

func (c *Collector) identifyBlockType(node *sitter.Node) (model.ElementKind, string) {
	parent := node.Parent()
	if parent == nil {
		return "", ""
	}
	pKind := parent.Kind()
	if pKind == "class_body" {
		return model.ScopeBlock, "$instance"
	}
	if pKind == "method_declaration" || pKind == "constructor_declaration" || pKind == "static_initializer" {
		return "", ""
	}
	return model.ScopeBlock, "block"
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

func (c *Collector) findInterfacesNode(node *sitter.Node) *sitter.Node {
	if n := node.ChildByFieldName("interfaces"); n != nil {
		return n
	}
	if n := node.ChildByFieldName("extends"); n != nil {
		return n
	}
	return c.findNamedChildOfType(node, "extends_interfaces")
}

// ==========================================
// 6. 原子辅助函数 (Atomic Helpers)
// ==========================================

func (c *Collector) isScopeContainer(k model.ElementKind, node *sitter.Node) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation,
		model.Method, model.Lambda, model.ScopeBlock, model.AnonymousClass:
		return true
	}
	return false
}

func (c *Collector) extractVariableName(node *sitter.Node, src *[]byte) string {
	if nNode := node.ChildByFieldName("name"); nNode != nil {
		return c.getNodeContent(nNode, *src)
	}
	if vd := c.findNamedChildOfType(node, "variable_declarator"); vd != nil {
		if nNode := vd.ChildByFieldName("name"); nNode != nil {
			return c.getNodeContent(nNode, *src)
		}
	}
	return ""
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

func (c *Collector) determineVariableKind(kindStr string) model.ElementKind {
	if kindStr == "local_variable_declaration" {
		return model.Variable
	}
	return model.Field
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

func (c *Collector) findDefinitionsByQN(fCtx *core.FileContext, qn string) []*core.DefinitionEntry {
	// 1. 先尝试直接从 QN 映射（如果你有这个 Map）
	// 2. 如果没有，则扫描全集（虽然慢，但最准）
	var result []*core.DefinitionEntry
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.Element.QualifiedName == qn {
				result = append(result, entry)
			}
		}
	}
	return result
}
