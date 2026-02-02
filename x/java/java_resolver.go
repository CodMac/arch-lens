package java

import (
	"strings"

	"github.com/CodMac/arch-lens/core"
	"github.com/CodMac/arch-lens/model"
)

type SymbolResolver struct{}

func NewJavaSymbolResolver() *SymbolResolver {
	return &SymbolResolver{}
}

func (j *SymbolResolver) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}

func (j *SymbolResolver) RegisterPackage(gc *core.GlobalContext, packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		if _, ok := gc.FindByQualifiedName(pkgQN); !ok {
			entry := core.DefinitionEntry{
				Element: &model.CodeElement{Kind: model.Package, Name: part, QualifiedName: pkgQN, IsFormSource: true},
			}
			gc.AddDefinition(&entry)
		}
	}
}

func (j *SymbolResolver) Resolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	gc.RLock()
	defer gc.RUnlock()

	// 1. 局部定义
	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}

	// 2. 精确导入
	if imps, ok := fc.Imports[symbol]; ok {
		for _, imp := range imps {
			// 仅import的源码符号才有可能在上下文中被发现，外部导入直接返回nil
			if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
				return []*core.DefinitionEntry{def}
			} else {
				return nil
			}
		}
	}

	// 3. 同包前缀
	pkgQN := j.BuildQualifiedName(fc.PackageName, symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	// 4. Java 特有的通配符导入
	for _, imps := range fc.Imports {
		for _, imp := range imps {
			if imp.IsWildcard {
				basePath := strings.TrimSuffix(imp.RawImportPath, "*")
				if def, ok := gc.FindByQualifiedName(basePath + symbol); ok {
					return []*core.DefinitionEntry{def}
				}
			}
		}
	}

	// 5. 直接按 QN 查找 (处理代码中使用全限定名调用的情况)
	if def, ok := gc.FindByQualifiedName(symbol); ok {
		return []*core.DefinitionEntry{def}
	}

	return nil
}
