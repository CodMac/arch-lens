package com.example.rel;

import java.util.function.Consumer;
import java.util.function.Supplier;

public class CaptureRelationSuite {

    private String fieldData = "field";
    private static int staticData = 100;

    public void testCaptures(String param) {
        String localVal = "local";

        // 1. Lambda 捕获局部变量
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$1
        // Target: com.example.rel.CaptureRelationSuite.testCaptures(String).localVal
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "localVal" }
        Runnable r1 = () -> System.out.println(localVal);

        // 2. Lambda 捕获方法参数
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$2
        // Target: com.example.rel.CaptureRelationSuite.testCaptures(String).param
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "param" }
        Consumer<String> c1 = (s) -> System.out.println(s + param);

        // 3. Lambda 捕获成员变量
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$3
        // Target: com.example.rel.CaptureRelationSuite.fieldData
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldData" }
        Supplier<String> s1 = () -> fieldData;

        // 4. Lambda 访问静态成员
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$4
        // Target: com.example.rel.CaptureRelationSuite.staticData
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "staticData" }
        Runnable r2 = () -> {
            int val = staticData;
        };

        // 5. 匿名内部类捕获局部变量
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).anonymousClass$1.run()
        // Target: com.example.rel.CaptureRelationSuite.testCaptures(String).localVal
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "localVal" }
        new Thread(new Runnable() {
            @Override
            public void run() {
                System.out.println(localVal);
            }
        }).start();

        // 6. 嵌套 Lambda 捕获
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$5.lambda$1
        // Target: com.example.rel.CaptureRelationSuite.testCaptures(String).localVal
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "localVal" }
        Runnable nested = () -> {
            Runnable inner = () -> System.out.println(localVal);
        };

        // 7. Lambda 对成员变量赋值 (Assign)
        // 注意：这里语法上是在 Lambda 内对 fieldData 赋值，
        // 但底层原理是捕获了 'this' 指针，然后通过 this.fieldData = "modified" 进行修改。
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).lambda$6
        // Target: com.example.rel.CaptureRelationSuite.fieldData
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldData" }
        Runnable r3 = () -> {
            fieldData = "modified";
        };

        // 8. 匿名内部类对成员变量赋值 (Assign)
        // 同样，匿名内部类持有外部类实例的引用（CaptureRelationSuite.this），因此可以赋值。
        // Source: com.example.rel.CaptureRelationSuite.testCaptures(String).anonymousClass$2.run()
        // Target: com.example.rel.CaptureRelationSuite.fieldData
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldData" }
        new Thread(new Runnable() {
            @Override
            public void run() {
                fieldData = "modifiedByAnon";
            }
        }).start();
    }
}