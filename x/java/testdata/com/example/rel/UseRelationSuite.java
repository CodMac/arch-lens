package com.example.rel;

import java.util.List;

public class UseRelationSuite {

    public static final String CONSTANT = "FIXED";
    private int fieldVar = 10;

    public void testUseCases(int param) {
        // 1. 局部变量读取 (Local Variable Use)
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).local)
        // Mores: { "java.rel.ast_kind": "identifier" }
        int local = 5;
        int result = local + 2;

        // 2. 成员变量读取 (Field Use - Explicit this)
        // Source: Method(testUseCases), Target: Field(com.example.rel.UseRelationSuite.fieldVar)
        // Mores: { "java.rel.use.receiver": "this", "java.rel.ast_kind": "identifier" }
        int x = this.fieldVar;

        // 3. 隐式成员变量与参数读取
        // Source: Method(testUseCases), Target: Field(com.example.rel.UseRelationSuite.fieldVar)
        // Source: Method(testUseCases), Target: Parameter(com.example.rel.UseRelationSuite.testUseCases(int).param)
        int y = fieldVar + param;

        // 4. 静态字段/常量访问 (Static Field Use)
        // Source: Method(testUseCases), Target: Field(com.example.rel.UseRelationSuite.CONSTANT)
        String s = UseRelationSuite.CONSTANT;

        // 5. 数组引用读取 (Array Access)
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).arr)
        int[] arr = {1, 2, 3};
        int val = arr[0];

        // 6. 方法参数传递 (Argument Use)
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).s)
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).x)
        print(s, x);

        // 7. 表达式/条件读取
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).x)
        boolean flag = (x > 0);

        // 8. 增强 for 循环中的集合读取
        List<String> list = List.of("A", "B");
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).list)
        for (String item : list) {
            // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).item)
            System.out.println(item);
        }

        // 9. Lambda 捕获读取 (Variable Capture)
        // Source: Lambda(com.example.rel.UseRelationSuite.testUseCases(int)$lambda$1), Target: Field(com.example.rel.UseRelationSuite.fieldVar)
        // Mores: { "java.rel.use.is_capture": true }
        Runnable r = () -> {
            System.out.println(fieldVar);
        };

        // 10. 类型强制转换中的读取 (Cast Operand Use)
        Object obj = "string";
        // Source: Method(testUseCases), Target: Variable(com.example.rel.UseRelationSuite.testUseCases(int).obj)
        String casted = (String) obj;
    }

    private void print(String s, int i) {}
}