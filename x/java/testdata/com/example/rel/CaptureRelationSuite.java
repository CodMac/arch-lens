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
        //   "java.rel.capture.kind": "local_variable",
        //   "java.rel.capture.is_effectively_final": true,
        //   "java.rel.ast_kind": "lambda_expression",
        //   "java.rel.raw_text": "localVal"
        // }
        Runnable r1 = () -> System.out.println(localVal);

        // 2. Lambda 捕获方法参数
        // Source: LambdaSymbol, Target: Parameter(param)
        // Mores: {
        //   "java.rel.capture.kind": "parameter",
        //   "java.rel.call.enclosing_method": "testCaptures",
        //   "java.rel.ast_kind": "lambda_expression"
        // }
        Consumer<String> c1 = (s) -> System.out.println(s + param);

        // 3. Lambda 捕获成员变量 (隐式通过 this 捕获)
        // Source: LambdaSymbol, Target: Field(fieldData)
        // Mores: {
        //   "java.rel.capture.kind": "field",
        //   "java.rel.call.receiver": "this",
        //   "java.rel.capture.is_implicit_this": true
        // }
        Supplier<String> s1 = () -> fieldData;

        // 4. Lambda 访问静态成员 (虽不属于 Heap 上的闭包捕获，但在依赖分析中需标记)
        // Source: LambdaSymbol, Target: Field(staticData)
        // Mores: {
        //   "java.rel.call.is_static": true,
        //   "java.rel.ast_kind": "lambda_expression"
        // }
        Runnable r2 = () -> staticData++;

        // 5. 匿名内部类捕获局部变量
        // Source: Method(run), Target: Variable(localVal)
        // Mores: {
        //   "java.rel.capture.kind": "local_variable",
        //   "java.rel.ast_kind": "anonymous_class_capture"
        // }
        new Thread(new Runnable() {
            @Override
            public void run() {
                // 这里发生了跨作用域访问
                System.out.println(localVal);
            }
        }).start();

        // 6. 嵌套 Lambda 捕获 (多级溯源)
        // Source: InnerLambdaSymbol, Target: Variable(localVal)
        // Mores: {
        //   "java.rel.capture.depth": 2,
        //   "java.rel.capture.enclosing_lambda": "OuterLambda",
        //   "java.rel.raw_text": "localVal"
        // }
        Runnable nested = () -> {
            Runnable inner = () -> System.out.println(localVal);
        };
    }
}