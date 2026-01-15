package com.example.rel;

import java.util.List;

public class UseRelationSuite {

    public static final String CONSTANT = "FIXED";
    private int fieldVar = 10;

    public void testUseCases(int param) {
        // 1. 局部变量读取 (Local Variable Use)
        // Source: Method(testUseCases), Target: Variable(local)
        // Mores: {
        //   "java.rel.use.parent_expression": "local + 2",
        //   "java.rel.use.usage_role": "operand",
        //   "java.rel.ast_kind": "identifier"
        // }
        int local = 5;
        int result = local + 2;

        // 2. 成员变量读取 (Field Use)
        // Source: Method(testUseCases), Target: Field(fieldVar)
        // Mores: {
        //   "java.rel.use.receiver": "this",
        //   "java.rel.ast_kind": "field_access"
        // }
        int x = this.fieldVar;

        // 3. 隐式成员变量与参数读取
        // Source: Method(testUseCases), Target: Field(fieldVar)
        // Mores: { "java.rel.use.usage_role": "operand", "java.rel.use.parent_expression": "fieldVar + param" }
        // Source: Method(testUseCases), Target: Parameter(param)
        // Mores: { "java.rel.use.usage_role": "operand" }
        int y = fieldVar + param;

        // 4. 静态字段/常量访问 (Static Field Use)
        // Source: Method(testUseCases), Target: Field(CONSTANT)
        // Mores: {
        //   "java.rel.use.receiver_type": "UseRelationSuite",
        //   "java.rel.use.is_static": true
        // }
        String s = UseRelationSuite.CONSTANT;

        // 5. 数组引用读取 (Array Access)
        // Source: Method(testUseCases), Target: Variable(arr)
        // Mores: {
        //   "java.rel.ast_kind": "array_access",
        //   "java.rel.use.index_expression": "0",
        //   "java.rel.use.usage_role": "array_source"
        // }
        int[] arr = {1, 2, 3};
        int val = arr[0];

        // 6. 方法参数传递 (Argument Use)
        // Source: Method(testUseCases), Target: Variable(s)
        // Mores: { "java.rel.use.call_site": "print", "java.rel.use.argument_index": 0 }
        // Source: Method(testUseCases), Target: Variable(x)
        // Mores: { "java.rel.use.call_site": "print", "java.rel.use.argument_index": 1 }
        print(s, x);

        // 7. 三元运算符/条件读取 (Conditional Use)
        // Source: Method(testUseCases), Target: Variable(x)
        // Mores: {
        //   "java.rel.ast_kind": "binary_expression",
        //   "java.rel.use.context": "if_condition"
        // }
        boolean flag = (x > 0);

        // 8. 增强 for 循环中的集合读取
        List<String> list = List.of("A", "B");
        // Source: Method(testUseCases), Target: Variable(list)
        // Mores: {
        //   "java.rel.ast_kind": "enhanced_for_statement",
        //   "java.rel.use.usage_role": "iterator_source"
        // }
        for (String item : list) {
            // Source: Method(testUseCases), Target: Variable(item)
            // Mores: { "java.rel.use.usage_role": "argument" }
            System.out.println(item);
        }

        // 9. Lambda 捕获读取 (Variable Capture)
        // Source: LambdaSymbol, Target: Field(fieldVar)
        // Mores: {
        //   "java.rel.use.is_capture": true,
        //   "java.rel.use.enclosing_method": "testUseCases"
        // }
        Runnable r = () -> {
            System.out.println(fieldVar);
        };

        // 10. 类型强制转换中的读取 (Cast Operand Use)
        Object obj = "string";
        // Source: Method(testUseCases), Target: Variable(obj)
        // Mores: {
        //   "java.rel.ast_kind": "cast_expression",
        //   "java.rel.use.target_type": "String"
        // }
        String casted = (String) obj;
    }

    private void print(String s, int i) {}
}