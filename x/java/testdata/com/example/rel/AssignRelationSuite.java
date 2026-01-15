package com.example.rel;

public class AssignRelationSuite {

    // 1. 字段声明初始化 (Field Initializer)
    // Source: Field(count), Target: Field(count)
    // Mores: { "java.rel.is_initializer": true, "java.rel.value_expression": "0", "java.rel.ast_kind": "variable_declarator" }
    // 注意：这里 Source 指向字段自身，表示该字段的初始值依赖逻辑。
    private int count = 0;

    private static String status;

    static {
        // 2. 静态代码块赋值 (Static Block Assignment)
        // Source: StaticInitializer(static {}), Target: Field(status)
        // Mores: { "java.rel.operator": "=", "java.rel.value_expression": "\"INIT\"", "java.rel.ast_kind": "assignment_expression" }
        status = "INIT";
    }

    public void testAssignments(int param) {
        // 3. 局部变量基础赋值 (Local Variable Assignment)
        // Source: Method(testAssignments), Target: Variable(local)
        // Mores: { "java.rel.operator": "=", "java.rel.value_expression": "10", "java.rel.ast_kind": "variable_declarator" }
        int local = 10;

        // Source: Method(testAssignments), Target: Variable(local)
        // Mores: { "java.rel.operator": "=", "java.rel.value_expression": "20" }
        local = 20;

        // 4. 成员变量赋值 (Field Assignment)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: { "java.rel.receiver": "this", "java.rel.operator": "=", "java.rel.value_expression": "100" }
        this.count = 100;

        // 5. 复合赋值 (Compound Assignment)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: { "java.rel.operator": "+=", "java.rel.value_expression": "5", "java.rel.is_compound": true }
        // 逻辑提示：这在底层隐含了一个对 count 的 USE 关系
        count += 5;

        // 6. 一次性多重赋值 (Multiple Assignment)
        int a, b, c;
        // Source: Method(testAssignments), Target: Variable(a)
        // Source: Method(testAssignments), Target: Variable(b)
        // Source: Method(testAssignments), Target: Variable(c)
        // Mores: { "java.rel.is_chained_assignment": true, "java.rel.value_expression": "50" }
        a = b = c = 50;

        // 7. 自增/自减 (Unary/Update Assignment)
        // Source: Method(testAssignments), Target: Field(count)
        // Mores: { "java.rel.operator": "++", "java.rel.is_postfix": true, "java.rel.ast_kind": "update_expression" }
        count++;

        // 8. 数组元素赋值 (Array Element Assignment)
        int[] arr = new int[5];
        // Source: Method(testAssignments), Target: Variable(arr)
        // Mores: { "java.rel.index_expression": "0", "java.rel.value_expression": "99", "java.rel.ast_kind": "assignment_expression" }
        arr[0] = 99;

        // 9. Lambda 表达式中的赋值 (Assignment in Lambda)
        // Source: LambdaSymbol, Target: Field(count)
        // Mores: { "java.rel.enclosing_method": "testAssignments", "java.rel.receiver": "this" }
        Runnable r = () -> {
            this.count = 300;
            int temp = 1;
            temp = 2; // Source: LambdaMethod, Target: Variable(temp)
        };
    }

    // 10. 构造函数中的属性赋值
    public AssignRelationSuite(int initialCount) {
        // Source: Constructor(AssignRelationSuite), Target: Field(count)
        // Mores: { "java.rel.operator": "=", "java.rel.value_expression": "initialCount" }
        this.count = initialCount;
    }
}