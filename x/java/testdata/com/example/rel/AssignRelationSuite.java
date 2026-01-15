package com.example.rel;

public class AssignRelationSuite {

    // 1. 字段声明初始化 (Field Initializer)
    // Source: Field(count), Target: Field(count)
    // Mores: {
    //   "java.rel.assign.is_initializer": true,
    //   "java.rel.assign.value_expression": "0",
    //   "java.rel.assign.operator": "=",
    //   "java.rel.ast_kind": "variable_declarator",
    //   "java.rel.raw_text": "count = 0"
    // }
    private int count = 0;

    private static String status;

    static {
        // 2. 静态代码块赋值
        // Source: StaticInitializer(static {}), Target: Field(status)
        // Mores: {
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "\"INIT\"",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.is_static_context": true
        // }
        status = "INIT";
    }

    public void testAssignments(int param) {
        // 3. 局部变量基础赋值 (声明时)
        // Source: Method(testAssignments), Target: Variable(local)
        // Mores: {
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "10",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.ast_kind": "variable_declarator"
        // }
        int local = 10;

        // 3.1 局部变量二次赋值 (表达式)
        // Source: Method(testAssignments), Target: Variable(local)
        // Mores: {
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "20",
        //   "java.rel.ast_kind": "assignment_expression"
        // }
        local = 20;

        // 4. 成员变量赋值 (带 Receiver)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: {
        //   "java.rel.call.receiver": "this",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "100"
        // }
        this.count = 100;

        // 5. 复合赋值 (Compound)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: {
        //   "java.rel.assign.operator": "+=",
        //   "java.rel.assign.value_expression": "5",
        //   "java.rel.assign.is_compound": true
        // }
        count += 5;

        // 6. 一次性多重赋值 (Chained)
        int a, b, c;
        // Source: Method(testAssignments), Target: Variable(a) ... (b), (c)
        // Mores: {
        //   "java.rel.assign.is_chained": true,
        //   "java.rel.assign.value_expression": "50",
        //   "java.rel.assign.operator": "="
        // }
        a = b = c = 50;

        // 7. 更新表达式 (Unary Update)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: {
        //   "java.rel.assign.operator": "++",
        //   "java.rel.assign.is_postfix": true,
        //   "java.rel.ast_kind": "update_expression"
        // }
        count++;

        // 8. 数组元素赋值
        int[] arr = new int[5];
        // Source: Method(testAssignments), Target: Variable(arr)
        // Mores: {
        //   "java.rel.assign.index_expression": "0",
        //   "java.rel.assign.value_expression": "99",
        //   "java.rel.ast_kind": "assignment_expression"
        // }
        arr[0] = 99;

        // 9. Lambda 内部赋值
        // Source: LambdaSymbol, Target: Field(count)
        // Mores: {
        //   "java.rel.call.enclosing_method": "testAssignments",
        //   "java.rel.call.receiver": "this",
        //   "java.rel.assign.operator": "="
        // }
        Runnable r = () -> {
            this.count = 300;
            int temp = 1;
            temp = 2;
        };
    }

    // 10. 构造函数赋值
    public AssignRelationSuite(int initialCount) {
        // Source: Constructor(AssignRelationSuite), Target: Field(count)
        // Mores: { "java.rel.assign.operator": "=", "java.rel.assign.value_expression": "initialCount" }
        this.count = initialCount;
    }
}