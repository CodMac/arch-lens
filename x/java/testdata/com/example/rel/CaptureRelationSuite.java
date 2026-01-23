package com.example.rel;

import java.util.function.Consumer;
import java.util.function.Supplier;

public class CaptureRelationSuite {

    private String fieldData = "field";
    private static int staticData = 100;

    public void testCaptures(String param) {
        String localVal = "local";

        // 1. Lambda 捕获局部变量
        // Source: Lambda(com.example.rel.CaptureRelationSuite.testCaptures(String)$lambda$1)
        // Target: Variable(com.example.rel.CaptureRelationSuite.testCaptures(String).localVal)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.ast_kind": "identifier" }
        Runnable r1 = () -> System.out.println(localVal);

        // 2. Lambda 捕获方法参数
        // Source: Lambda(com.example.rel.CaptureRelationSuite.testCaptures(String)$lambda$2)
        // Target: Parameter(com.example.rel.CaptureRelationSuite.testCaptures(String).param)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.call.enclosing_method": "com.example.rel.CaptureRelationSuite.testCaptures(String)" }
        Consumer<String> c1 = (s) -> System.out.println(s + param);

        // 3. Lambda 捕获成员变量 (通过 implicit this 访问)
        // Source: Lambda(com.example.rel.CaptureRelationSuite.testCaptures(String)$lambda$3)
        // Target: Field(com.example.rel.CaptureRelationSuite.fieldData)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.use.receiver": "this" }
        Supplier<String> s1 = () -> fieldData;

        // 4. Lambda 访问静态成员 (虽不属于 Heap 闭包捕获，但属跨作用域 Use)
        // Source: Lambda(com.example.rel.CaptureRelationSuite.testCaptures(String)$lambda$4)
        // Target: Field(com.example.rel.CaptureRelationSuite.staticData)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.ast_kind": "identifier" }
        Runnable r2 = () -> {
            int val = staticData;
        };

        // 5. 匿名内部类捕获局部变量
        // Source: Method(com.example.rel.CaptureRelationSuite$1.run)
        // Target: Variable(com.example.rel.CaptureRelationSuite.testCaptures(String).localVal)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.call.enclosing_method": "com.example.rel.CaptureRelationSuite.testCaptures(String)" }
        new Thread(new Runnable() {
            @Override
            public void run() {
                System.out.println(localVal);
            }
        }).start();

        // 6. 嵌套 Lambda 捕获
        // Source: Lambda(...$lambda$5$lambda$1)
        // Target: Variable(com.example.rel.CaptureRelationSuite.testCaptures(String).localVal)
        // Mores: { "java.rel.use.is_capture": true, "java.rel.raw_text": "localVal" }
        Runnable nested = () -> {
            Runnable inner = () -> System.out.println(localVal);
        };
    }
}