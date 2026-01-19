package com.example.rel;

public class AssignRelationSuite {

    // 1. 字段声明初始化 (Field Initializer)
    // Source: com.example.rel.AssignRelationSuite.count (FIELD)
    // Target: com.example.rel.AssignRelationSuite.count (FIELD)
    // Mores: {
    //   "java.rel.raw_text": "count = 0",
    //   "java.rel.context": "variable_declarator",
    //   "java.rel.ast_kind": "variable_declarator",
    //   "java.rel.assign.target_name": "count",
    //   "java.rel.assign.is_initializer": true,
    //   "java.rel.assign.operator": "=",
    //   "java.rel.assign.value_expression": "0"
    // }
    private int count = 0;

    private static String status;

    static {
        // 2. 静态代码块赋值
        // Source: com.example.rel.AssignRelationSuite.$static$1 (SCOPE_BLOCK)
        // Target: com.example.rel.AssignRelationSuite.status (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "status = \"INIT\"",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "status",
        //   "java.rel.assign.is_static_context": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "\"INIT\""
        // }
        status = "INIT";
    }

    public void testAssignments(int param) {
        // 3. 局部变量基础赋值 (声明时)
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.testAssignments(int).local (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "local = 10",
        //   "java.rel.context": "variable_declarator",
        //   "java.rel.ast_kind": "variable_declarator",
        //   "java.rel.assign.target_name": "local",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "10"
        // }
        int local = 10;

        // 3.1 局部变量二次赋值 (表达式)
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.testAssignments(int).local (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "local = 20",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "local",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "20"
        // }
        local = 20;

        // 4. 成员变量赋值 (带 Receiver)
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.count (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "this.count = 100",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "count",
        //   "java.rel.assign.receiver": "this",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "100"
        // }
        this.count = 100;

        // 5. 复合赋值 (Compound)
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.count (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "count += 5",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "count",
        //   "java.rel.assign.operator": "+=",
        //   "java.rel.assign.value_expression": "5",
        //   "java.rel.assign.is_compound": true
        // }
        count += 5;

        // 6. 一次性多重赋值 (Chained)
        int a, b, c;
        // 关系 A -> a
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.testAssignments(int).a (VARIABLE)
        // Mores: { "java.rel.assign.is_chained": true, "java.rel.assign.value_expression": "b = c = 50", ... }

        // 关系 A -> b
        // Target: b (VARIABLE)
        // Mores: { "java.rel.assign.is_chained": true, "java.rel.assign.value_expression": "c = 50", ... }

        // 关系 A -> c
        // Target: c (VARIABLE)
        // Mores: { "java.rel.assign.value_expression": "50", ... }
        a = b = c = 50;

        // 7. 更新表达式 (Unary Update)
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.count (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "count++",
        //   "java.rel.context": "update_expression",
        //   "java.rel.ast_kind": "update_expression",
        //   "java.rel.assign.target_name": "count",
        //   "java.rel.assign.operator": "++",
        //   "java.rel.assign.is_postfix": true
        // }
        count++;
        --count;

        // 8. 数组元素赋值
        int[] arr = new int[5];
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationSuite.testAssignments(int).arr (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "arr[0] = 99",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "arr",
        //   "java.rel.assign.index_expression": "0",
        //   "java.rel.assign.value_expression": "99",
        //   "java.rel.assign.operator": "="
        // }
        arr[0] = 99;

        // 9. Lambda 内部赋值
        // Source: com.example.rel.AssignRelationSuite.testAssignments(int).lambda$1 (LAMBDA)
        // Target: com.example.rel.AssignRelationSuite.count (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "this.count = 300",
        //   "java.rel.context": "assignment_expression",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "count",
        //   "java.rel.assign.receiver": "this",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "300"
        // }
        Runnable r = () -> {
            this.count = 300;
            // 内部变量定义
            // Source: ...lambda$1 (LAMBDA), Target: ...lambda$1.temp (VARIABLE)
            int temp = 1;
            temp = 2;
        };
    }

    // 10. 构造函数赋值
    // Source: com.example.rel.AssignRelationSuite.AssignRelationSuite(int) (METHOD)
    // Target: com.example.rel.AssignRelationSuite.count (FIELD)
    // Mores: {
    //   "java.rel.raw_text": "this.count = initialCount",
    //   "java.rel.context": "assignment_expression",
    //   "java.rel.ast_kind": "assignment_expression",
    //   "java.rel.assign.target_name": "count",
    //   "java.rel.assign.receiver": "this",
    //   "java.rel.assign.operator": "=",
    //   "java.rel.assign.value_expression": "initialCount"
    // }
    public AssignRelationSuite(int initialCount) {
        this.count = initialCount;
    }
}