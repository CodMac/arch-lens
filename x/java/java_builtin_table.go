package java

import "github.com/CodMac/go-treesitter-dependency-analyzer/model"

// --- Java 内置符号表 ---

var BuiltinTable = map[string]struct {
	QN   string
	Kind model.ElementKind
}{
	// === java.lang 核心类 (默认隐式导入) ===
	"String":              {"java.lang.String", model.Class},
	"Object":              {"java.lang.Object", model.Class},
	"System":              {"java.lang.System", model.Class},
	"Integer":             {"java.lang.Integer", model.Class},
	"Long":                {"java.lang.Long", model.Class},
	"Double":              {"java.lang.Double", model.Class},
	"Float":               {"java.lang.Float", model.Class},
	"Boolean":             {"java.lang.Boolean", model.Class},
	"Byte":                {"java.lang.Byte", model.Class},
	"Character":           {"java.lang.Character", model.Class},
	"Short":               {"java.lang.Short", model.Class},
	"Void":                {"java.lang.Void", model.Class},
	"Number":              {"java.lang.Number", model.Class},
	"Math":                {"java.lang.Math", model.Class},
	"Class":               {"java.lang.Class", model.Class},
	"ClassLoader":         {"java.lang.ClassLoader", model.Class},
	"Thread":              {"java.lang.Thread", model.Class},
	"ThreadGroup":         {"java.lang.ThreadGroup", model.Class},
	"ThreadLocal":         {"java.lang.ThreadLocal", model.Class},
	"StringBuilder":       {"java.lang.StringBuilder", model.Class},
	"StringBuffer":        {"java.lang.StringBuffer", model.Class},
	"Enum":                {"java.lang.Enum", model.Class},
	"Throwable":           {"java.lang.Throwable", model.Class},
	"Exception":           {"java.lang.Exception", model.Class},
	"RuntimeException":    {"java.lang.RuntimeException", model.Class},
	"Error":               {"java.lang.Error", model.Class},
	"StackTraceElement":   {"java.lang.StackTraceElement", model.Class},
	"Iterable":            {"java.lang.Iterable", model.Interface},
	"AutoCloseable":       {"java.lang.AutoCloseable", model.Interface},
	"Runnable":            {"java.lang.Runnable", model.Interface},
	"Comparable":          {"java.lang.Comparable", model.Interface},
	"CharSequence":        {"java.lang.CharSequence", model.Interface},
	"Override":            {"java.lang.Override", model.KAnnotation},
	"Deprecated":          {"java.lang.Deprecated", model.KAnnotation},
	"SuppressWarnings":    {"java.lang.SuppressWarnings", model.KAnnotation},
	"SafeVarargs":         {"java.lang.SafeVarargs", model.KAnnotation},
	"FunctionalInterface": {"java.lang.FunctionalInterface", model.KAnnotation},

	// === java.lang 常用异常 ===
	"NullPointerException":          {"java.lang.NullPointerException", model.Class},
	"IllegalArgumentException":      {"java.lang.IllegalArgumentException", model.Class},
	"IllegalStateException":         {"java.lang.IllegalStateException", model.Class},
	"IndexOutOfBoundsException":     {"java.lang.IndexOutOfBoundsException", model.Class},
	"UnsupportedOperationException": {"java.lang.UnsupportedOperationException", model.Class},

	// === java.lang.annotation 核心元注解与枚举 ===
	"Retention":     {"java.lang.annotation.Retention", model.KAnnotation},
	"Target":        {"java.lang.annotation.Target", model.KAnnotation},
	"Documented":    {"java.lang.annotation.Documented", model.KAnnotation},
	"Inherited":     {"java.lang.annotation.Inherited", model.KAnnotation},
	"Native":        {"java.lang.annotation.Native", model.KAnnotation},
	"Repeatable":    {"java.lang.annotation.Repeatable", model.KAnnotation},
	"Resource":      {"javax.annotation.Resource", model.KAnnotation},
	"PostConstruct": {"javax.annotation.PostConstruct", model.KAnnotation},
	"PreDestroy":    {"javax.annotation.PreDestroy", model.KAnnotation},
	"Generated":     {"javax.annotation.Generated", model.KAnnotation},
	"Nullable":      {"javax.annotation.Nullable", model.KAnnotation},
	"Nonnull":       {"javax.annotation.Nonnull", model.KAnnotation},

	// 元注解使用的枚举类型
	"RetentionPolicy": {"java.lang.annotation.RetentionPolicy", model.Enum},
	"ElementType":     {"java.lang.annotation.ElementType", model.Enum},

	// 常见的枚举常量 (支持在注解参数中直接解析)
	"RUNTIME":   {"java.lang.annotation.RetentionPolicy.RUNTIME", model.Field},
	"SOURCE":    {"java.lang.annotation.RetentionPolicy.SOURCE", model.Field},
	"CLASS":     {"java.lang.annotation.RetentionPolicy.CLASS", model.Field},
	"TYPE":      {"java.lang.annotation.ElementType.TYPE", model.Field},
	"METHOD":    {"java.lang.annotation.ElementType.METHOD", model.Field},
	"FIELD":     {"java.lang.annotation.ElementType.FIELD", model.Field},
	"PARAMETER": {"java.lang.annotation.ElementType.PARAMETER", model.Field},

	// === java.util 集合框架 ===
	"Collection":    {"java.util.Collection", model.Interface},
	"List":          {"java.util.List", model.Interface},
	"ArrayList":     {"java.util.ArrayList", model.Class},
	"LinkedList":    {"java.util.LinkedList", model.Class},
	"Set":           {"java.util.Set", model.Interface},
	"HashSet":       {"java.util.HashSet", model.Class},
	"TreeSet":       {"java.util.TreeSet", model.Class},
	"Map":           {"java.util.Map", model.Interface},
	"HashMap":       {"java.util.HashMap", model.Class},
	"TreeMap":       {"java.util.TreeMap", model.Class},
	"LinkedHashMap": {"java.util.LinkedHashMap", model.Class},
	"Iterator":      {"java.util.Iterator", model.Interface},
	"Optional":      {"java.util.Optional", model.Class},
	"Arrays":        {"java.util.Arrays", model.Class},
	"Collections":   {"java.util.Collections", model.Class},
	"UUID":          {"java.util.UUID", model.Class},
	"Date":          {"java.util.Date", model.Class},
	"Objects":       {"java.util.Objects", model.Class},
	"Scanner":       {"java.util.Scanner", model.Class},
	"Properties":    {"java.util.Properties", model.Class},

	// === java.util.stream & function (现代 Java 高频) ===
	"Stream":     {"java.util.stream.Stream", model.Interface},
	"Collectors": {"java.util.stream.Collectors", model.Class},
	"Function":   {"java.util.function.Function", model.Interface},
	"BiFunction": {"java.util.function.BiFunction", model.Interface},
	"Consumer":   {"java.util.function.Consumer", model.Interface},
	"Predicate":  {"java.util.function.Predicate", model.Interface},
	"Supplier":   {"java.util.function.Supplier", model.Interface},

	// === java.time (JSR-310 现代日期) ===
	"LocalDate":     {"java.time.LocalDate", model.Class},
	"LocalTime":     {"java.time.LocalTime", model.Class},
	"LocalDateTime": {"java.time.LocalDateTime", model.Class},
	"ZonedDateTime": {"java.time.ZonedDateTime", model.Class},
	"Duration":      {"java.time.Duration", model.Class},
	"Instant":       {"java.time.Instant", model.Class},

	// === java.io & java.nio ===
	"InputStream":  {"java.io.InputStream", model.Class},
	"OutputStream": {"java.io.OutputStream", model.Class},
	"File":         {"java.io.File", model.Class},
	"Serializable": {"java.io.Serializable", model.Interface},
	"Path":         {"java.nio.file.Path", model.Interface},
	"Paths":        {"java.nio.file.Paths", model.Class},
	"Files":        {"java.nio.file.Files", model.Class},

	// === java.util.concurrent ===
	"Executor":          {"java.util.concurrent.Executor", model.Interface},
	"ExecutorService":   {"java.util.concurrent.ExecutorService", model.Interface},
	"Executors":         {"java.util.concurrent.Executors", model.Class},
	"Future":            {"java.util.concurrent.Future", model.Interface},
	"CompletableFuture": {"java.util.concurrent.CompletableFuture", model.Class},
	"ConcurrentHashMap": {"java.util.concurrent.ConcurrentHashMap", model.Class},
	"TimeUnit":          {"java.util.concurrent.TimeUnit", model.Enum},

	// === 静态字段与内置对象 ===
	"out": {"java.lang.System.out", model.Field},
	"err": {"java.lang.System.err", model.Field},
	"in":  {"java.lang.System.in", model.Field},
}
