package com.example.rel;

import java.util.List;

public class UseRelationSuite {

    public static final String CONSTANT = "FIXED";
    private int fieldVar = 10;

    public void testUseCases(int param) {
        // 1. 局部变量读取 (Local Variable Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).local
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "local", "java.rel.context": "binary_expression" }
        int local = 5;
        int result = local + 2;

        // 2. 成员变量读取 (Field Use - Explicit this)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.fieldVar
        // Mores: { "java.rel.use.receiver": "this", "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldVar" }
        int x = this.fieldVar;

        // 3. 隐式成员变量与参数读取
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.fieldVar
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).param
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldVar" }
        int y = fieldVar + param;

        // 4. 静态字段/常量访问 (Static Field Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.CONSTANT
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "CONSTANT" }
        String s = UseRelationSuite.CONSTANT;

        // 5. 数组引用读取 (Array Access)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).arr
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.raw_text": "arr", "java.rel.context": "array_access" }
        int[] arr = {1, 2, 3};
        int val = arr[0];

        // 6. 方法参数传递 (Argument Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).s
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).x
        // Mores: { "java.rel.context": "argument_list", "java.rel.raw_text": "s" }
        print(s, x);

        // 7. 表达式/条件读取
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).x
        // Mores: { "java.rel.context": "parenthesized_expression", "java.rel.ast_kind": "identifier", "java.rel.raw_text": "x" }
        boolean flag = (x > 0);

        // 8. 增强 for 循环中的集合读取
        List<String> list = List.of("A", "B");
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).list
        // Mores: { "java.rel.context": "enhanced_for_statement", "java.rel.ast_kind": "identifier", "java.rel.raw_text": "list" }
        for (String item : list) {
            // Source: com.example.rel.UseRelationSuite.testUseCases(int)
            // Target: com.example.rel.UseRelationSuite.testUseCases(int).item
            // Mores: { "java.rel.context": "method_invocation", "java.rel.raw_text": "item" }
            System.out.println(item);
        }

        // 9. Lambda 捕获读取 (Variable Capture)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int).lambda$1
        // Target: com.example.rel.UseRelationSuite.fieldVar
        // Mores: { "java.rel.use.is_capture": true, "java.rel.ast_kind": "identifier", "java.rel.raw_text": "fieldVar" }
        Runnable r = () -> {
            System.out.println(fieldVar);
        };

        // 10. 类型强制转换中的读取 (Cast Operand Use)
        Object obj = "string";
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).obj
        // Mores: { "java.rel.context": "cast_expression", "java.rel.raw_text": "obj" }
        String casted = (String) obj;
    }

    private void print(String s, int i) {}
}