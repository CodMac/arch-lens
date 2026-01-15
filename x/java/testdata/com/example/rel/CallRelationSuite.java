package com.example.rel;

import java.util.ArrayList;
import java.util.List;
import java.util.function.Consumer;

public class CallRelationSuite extends BaseClass {

    private static String staticField = "test";

    public void executeAll() {
        // 1. 基础实例调用 (Local Instance Call)
        // Source: Method(executeAll), Target: Method(simpleMethod)
        // Mores: { "java.rel.receiver": "this", "java.rel.raw_text": "simpleMethod()" }
        simpleMethod();

        // 2. 静态方法调用 (Static Call)
        // Source: Method(executeAll), Target: Method(staticMethod)
        // Mores: { "java.rel.is_static": true, "java.rel.receiver_type": "CallRelationSuite" }
        CallRelationSuite.staticMethod();

        // 3. 跨包静态调用 (Cross-package Static Call)
        // Source: Method(executeAll), Target: Method(currentTimeMillis)
        // Mores: { "java.rel.receiver_type": "java.lang.System", "java.rel.ast_kind": "method_invocation" }
        System.currentTimeMillis();

        // 4. 链式调用 (Chained Call)
        // Source: Method(executeAll), Target: Method(add)
        // Mores: { "java.rel.receiver_expression": "getList()", "java.rel.ast_kind": "method_invocation" }
        // 注意：这里还会触发一个 [executeAll -> getList] 的 CALL 关系
        getList().add("item");

        // 5. 显式继承调用 (Explicit Super Call)
        // Source: Method(executeAll), Target: Method(BaseClass.baseMethod)
        // Mores: { "java.rel.receiver": "super", "java.rel.raw_text": "super.baseMethod()" }
        super.baseMethod();

        // 6. 隐式继承调用 (Implicit Super Call)
        // Source: Method(executeAll), Target: Method(BaseClass.baseMethod)
        // Mores: { "java.rel.receiver": "this", "java.rel.is_inherited": true }
        baseMethod();

        // 7. 构造函数调用 (Constructor Call / CREATE)
        // Source: Method(executeAll), Target: Class(ArrayList)
        // Mores: { "java.rel.ast_kind": "object_creation_expression", "java.rel.is_generic": true }
        List<String> list = new ArrayList<>();

        // 8. Lambda 内部的方法调用 (Call within Lambda)
        // Source: LambdaSymbol(s -> ...), Target: Method(simpleMethod)
        // Mores: { "java.rel.enclosing_method": "executeAll", "java.rel.ast_kind": "method_invocation" }
        Consumer<String> consumer = (s) -> {
            simpleMethod();
            // 同时触发 CAPTURE 关系: [Lambda -> Field(staticField)]
            System.out.println(s + staticField);
        };

        // 9. 方法引用 (Method Reference)
        // Source: Method(executeAll), Target: Method(simpleMethod)
        // Mores: { "java.rel.ast_kind": "method_reference", "java.rel.receiver": "this" }
        List.of("A").forEach(this::simpleMethod);

        // 10. 泛型方法显式调用 (Generic Method Call)
        // Source: Method(executeAll), Target: Method(genericMethod)
        // Mores: { "java.rel.type_arguments": "String", "java.rel.raw_text": "this.<String>genericMethod(\"hello\")" }
        this.<String>genericMethod("hello");

        // 11. 匿名内部类调用 (Anonymous Inner Class Call)
        // Source: AnonymousClassSymbol, Target: Method(run)
        // Mores: { "java.rel.ast_kind": "method_invocation", "java.rel.parent_type": "Runnable" }
        new Runnable() {
            @Override
            public void run() {
                // Source: Method(run), Target: Method(simpleMethod)
                simpleMethod();
            }
        }.run();
    }

    public void simpleMethod() {}
    public void simpleMethod(String s) {}
    public static void staticMethod() {}
    public List<String> getList() { return new ArrayList<>(); }
    public <T> void genericMethod(T t) {}

    class SubClass {
        SubClass() {
            // Source: Constructor(SubClass), Target: Constructor(BaseClass)
            // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.receiver": "super" }
            super();
        }
        SubClass(String data) {
            // Source: Constructor(SubClass), Target: Constructor(SubClass)
            // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.receiver": "this" }
            this();
        }
    }
}

class BaseClass {
    public void baseMethod() {}
}