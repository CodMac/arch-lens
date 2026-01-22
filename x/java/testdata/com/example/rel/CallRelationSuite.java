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
        //   "java.rel.raw_text": "simpleMethod()",
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.receiver": "this",
        //   "java.rel.call.is_static": false
        // }
        simpleMethod();

        // 2. 静态方法调用 (类名在当前包)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.CallRelationSuite.staticMethod() (METHOD)
        // Mores: {
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.is_static": true,
        //   "java.rel.call.receiver_type": "CallRelationSuite"
        // }
        CallRelationSuite.staticMethod();

        // 3. 非源码静态调用 (未显式 import，保持原样)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: System.currentTimeMillis (SYMBOL)
        // Mores: {
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.is_static": true,
        //   "java.rel.call.receiver_type": "System"
        // }
        System.currentTimeMillis();

        // 4. 链式调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: add (SYMBOL)
        // Mores: {
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.is_chained": true
        // }
        getList().add("item");

        // 5. 显式继承调用 (Super)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.BaseClass.baseMethod() (METHOD)
        // Mores: {
        //   "java.rel.raw_text": "super.baseMethod()",
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.receiver": "super"
        // }
        super.baseMethod();

        // 6. 隐式继承调用
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: com.example.rel.BaseClass.baseMethod() (METHOD)
        // Mores: {
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.receiver": "this"
        // }
        baseMethod();

        // 7. 对象创建 (在 import 清单中，补全 QN)
        // Source: com.example.rel.CallRelationSuite.executeAll() (METHOD)
        // Target: java.util.ArrayList.ArrayList() (METHOD)
        // Mores: {
        //   "java.rel.ast_kind": "object_creation_expression",
        //   "java.rel.call.is_constructor": true
        // }
        List<String> list = new ArrayList<>();

        // 8. Lambda 内部的方法调用
        // Source: com.example.rel.CallRelationSuite.executeAll().lambda (LAMBDA)
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: {
        //   "java.rel.ast_kind": "method_invocation",
        //   "java.rel.call.enclosing_method": "com.example.rel.CallRelationSuite.executeAll",
        //   "java.rel.call.receiver": "this"
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
        //   "java.rel.raw_text": "this.<String>genericMethod(\"hello\")",
        //   "java.rel.ast_kind": "method_invocation"
        // }
        this.<String>genericMethod("hello");

        // 11. 匿名内部类调用
        new Runnable() {
            @Override
            public void run() {
                // Source: com.example.rel.CallRelationSuite$1.run() (METHOD)
                // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
                // Mores: {
                //   "java.rel.ast_kind": "method_invocation",
                //   "java.rel.call.enclosing_method": "com.example.rel.CallRelationSuite.executeAll"
                // }
                simpleMethod();
            }
        }.run();

        // 12. 可变参数调用
        // Target: com.example.rel.CallRelationSuite.customLog(String, Object[]) (METHOD)
        customLog("log", 1, 2);

        // 13. 强制类型转换后的调用
        Object obj = new CallRelationSuite();
        // Target: com.example.rel.CallRelationSuite.simpleMethod() (METHOD)
        // Mores: { "java.rel.call.receiver": "obj" }
        ((CallRelationSuite)obj).simpleMethod();

        // 14. 异常抛出 (非源码，未 import)
        // Target: RuntimeException (SYMBOL)
        // Mores: { "java.rel.call.is_constructor": true, "java.rel.ast_kind": "object_creation_expression" }
        if (staticField == null) {
            throw new RuntimeException("Error");
        }

        // 15. 枚举静态方法调用
        // Target: com.example.rel.CallRelationSuite.MyEnum.values() (METHOD)
        // Mores: { "java.rel.call.is_static": true }
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
            // Source: com.example.rel.CallRelationSuite.SubClass.SubClass() (METHOD)
            // Target: com.example.rel.BaseClass.BaseClass() (METHOD)
            // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.call.is_constructor": true }
            super();
        }
    }
}

class BaseClass {
    public BaseClass() {}
    public void baseMethod() {}
}