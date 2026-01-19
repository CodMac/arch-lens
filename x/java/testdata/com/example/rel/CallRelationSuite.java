package com.example.rel;

import java.util.ArrayList;
import java.util.List;
import java.util.function.Consumer;

public class CallRelationSuite extends BaseClass {

    private static String staticField = "test";

    public void executeAll() {
        // 1. 基础实例调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.receiver": "this",
        //   "java.rel.call.is_static": false,
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.raw_text": "simpleMethod()"
        // }
        simpleMethod();

        // 2. 静态方法调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.staticMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.is_static": true,
        //   "java.rel.call.receiver_type": "CallRelationSuite",
        //   "java.rel.ast_kind": "method_invocation"
        // }
        CallRelationSuite.staticMethod();

        // 3. 跨包静态调用 (第三方库/JDK)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: System.currentTimeMillis (SYMBOL)
        // Mores: {
        //   "java.rel.call.receiver_type": "System",
        //   "java.rel.ast_kind": "method_invocation"
        // }
        System.currentTimeMillis();

        // 4. 链式调用 (Chained Call)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: add (SYMBOL)  // 假设 add 是外部 List 的方法
        // Mores: {
        //   "java.rel.call.receiver_expression": "getList()",
        //   "java.rel.call.is_chained": true,
        //   "java.rel.ast_kind": "method_invocation"
        // }
        // 注意：此处解析引擎应同时识别出 executeAll -> getList() 的调用
        getList().add("item");

        // 5. 显式继承调用 (Super)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.BaseClass.baseMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.receiver": "super",
        //   "java.rel.call.is_inherited": true,
        //   "java.rel.raw_text": "super.baseMethod()"
        // }
        super.baseMethod();

        // 6. 隐式继承调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.BaseClass.baseMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.receiver": "this",
        //   "java.rel.call.is_inherited": true
        // }
        baseMethod();

        // 7. 对象创建 (Constructor Call)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: ArrayList (SYMBOL)
        // Mores: {
        //   "java.rel.ast_kind": "object_creation_expression",
        //   "java.rel.call.is_constructor": true
        // }
        List<String> list = new ArrayList<>();

        // 8. Lambda 内部的方法调用
        // Source: com.example.rel.CallRelationSuite.executeAll().lambda$1 (LAMBDA)
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.enclosing_method": "executeAll",
        //   "java.rel.call.receiver": "this",
        //   "java.rel.ast_kind": "method_invocation"
        // }
        Consumer<String> consumer = (s) -> {
            simpleMethod();
        };

        // 9. 方法引用 (Method Reference)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: {
        //   "java.rel.ast_kind": "method_reference",
        //   "java.rel.call.receiver": "this",
        //   "java.rel.call.is_functional": true
        // }
        List.of("A").forEach(this::simpleMethod);

        // 10. 泛型方法显式调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.genericMethod(T) (METHOD)
        // Mores: {
        //   "java.rel.call.type_arguments": "String",
        //   "java.rel.raw_text": "this.<String>genericMethod(\"hello\")"
        // }
        this.<String>genericMethod("hello");

        // 11. 匿名内部类调用
        // Source: com.example.rel.CallRelationSuite.executeAll().$1 (CLASS)
        // Target: Runnable.run (SYMBOL)
        // Mores: { "java.rel.ast_kind": "method_invocation", "java.rel.call.is_anonymous_class": true }
        new Runnable() {
            @Override
            public void run() {
                // Source: ...$1.run() (METHOD), Target: ...simpleMethod() (METHOD)
                simpleMethod();
            }
        }.run();

        // 12. 增强：自定义可变参数调用 (Varargs)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.customLog(String, Object...) (METHOD)
        // Mores: {
        //   "java.rel.call.is_varargs": true,
        //   "java.rel.call.args_count": 3,
        //   "java.rel.raw_text": "customLog(\"log\", 1, 2)"
        // }
        customLog("log", 1, 2);

        // 13. 增强：强制类型转换调用 (Casted Receiver)
        Object obj = new CallRelationSuite();
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: {
        //   "java.rel.call.receiver": "obj",
        //   "java.rel.call.receiver_cast_type": "CallRelationSuite",
        //   "java.rel.ast_kind": "method_invocation"
        // }
        ((CallRelationSuite)obj).simpleMethod();

        // 14. 增强：异常抛出 (Constructor call for Exception)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: RuntimeException (SYMBOL)
        // Mores: { "java.rel.call.is_constructor": true, "java.rel.ast_kind": "object_creation_expression" }
        if (staticField == null) {
            throw new RuntimeException("Error");
        }

        // 15. 增强：枚举隐式方法调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.MyEnum.values() (METHOD)
        // Mores: { "java.rel.call.is_static": true, "java.rel.call.is_implicit": true }
        MyEnum[] tags = MyEnum.values();
    }

    public enum MyEnum { TAG1, TAG2 }
    public void simpleMethod() {}
    public static void staticMethod() {}
    public List<String> getList() { return new ArrayList<>(); }
    public <T> void genericMethod(T t) {}
    public void customLog(String prefix, Object... args) {}

    class SubClass extends BaseClass {
        SubClass() {
            // 16. 显式构造函数调用 (Super)
            // Source: com.example.rel.CallRelationSuite.SubClass.SubClass() (METHOD)
            // Target: com.example.rel.BaseClass.BaseClass() (METHOD)
            // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.call.receiver": "super" }
            super();
        }
    }
}

class BaseClass {
    public BaseClass() {}
    public void baseMethod() {}
}