package com.example.rel;

import java.util.function.Consumer;
import java.util.function.Supplier;

public class CaptureRelationSuite {

    private String fieldData = "field";
    private static int staticData = 100;

    public void testCaptures(String param) {
        String localVal = "local";

        // 1. Lambda 捕获局部变量
        // Source: LambdaSymbol, Target: Variable(localVal)
        // Mores: {
        //   "java.rel.capture_kind": "local_variable",
        //   "java.rel.is_effectively_final": true,
        //   "java.rel.raw_text": "localVal"
        // }
        Runnable r1 = () -> System.out.println(localVal);

        // 2. Lambda 捕获方法参数
        // Source: LambdaSymbol, Target: Parameter(param)
        // Mores: { "java.rel.capture_kind": "parameter", "java.rel.enclosing_method": "testCaptures" }
        Consumer<String> c1 = (s) -> System.out.println(s + param);

        // 3. Lambda 捕获成员变量 (隐式通过 this 捕获)
        // Source: LambdaSymbol, Target: Field(fieldData)
        // Mores: { "java.rel.capture_kind": "field", "java.rel.receiver": "this" }
        Supplier<String> s1 = () -> fieldData;

        // 4. Lambda 捕获静态成员 (不算严格意义上的闭包捕获，但在依赖分析中常记为 CAPTURE 或 USE)
        // Source: LambdaSymbol, Target: Field(staticData)
        // Mores: { "java.rel.is_static": true }
        Runnable r2 = () -> staticData++;

        // 5. 匿名内部类捕获局部变量
        // Source: Method(run), Target: Variable(localVal)
        // Mores: { "java.rel.ast_kind": "anonymous_class_capture" }
        new Thread(new Runnable() {
            @Override
            public void run() {
                System.out.println(localVal);
            }
        }).start();

        // 6. 嵌套 Lambda 捕获
        // Source: InnerLambdaSymbol, Target: Variable(localVal)
        // Mores: { "java.rel.capture_depth": 2 }
        Runnable nested = () -> {
            Runnable inner = () -> System.out.println(localVal);
        };
    }
}