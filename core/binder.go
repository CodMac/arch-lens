package core

import "fmt"

// Binder 用于符号消解和元数据增强（语义绑定）
type Binder interface {
	// BindSymbols 在全局符号采集完成后，将原始类型文本绑定为全限定名(QN)
	BindSymbols(gc *GlobalContext)
}

var binderMap = make(map[Language]Binder)

// RegisterBinder 注册一个语言与其对应的 Binder
func RegisterBinder(lang Language, binder Binder) {
	binderMap[lang] = binder
}

// GetBinder 根据语言类型获取对应的 Binder 实例。
func GetBinder(lang Language) (Binder, error) {
	binder, ok := binderMap[lang]
	if !ok {
		return nil, fmt.Errorf("no binder registered for language: %s", lang)
	}

	return binder, nil
}
