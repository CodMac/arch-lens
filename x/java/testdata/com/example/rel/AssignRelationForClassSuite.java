package com.example.rel;

public class AssignRelationForClassSuite {
    // 字段声明初始化
    // Source: com.example.rel.AssignRelationForClassSuite.count (FIELD)
    // Target: com.example.rel.AssignRelationForClassSuite.count (FIELD)
    // Mores: { "is_initializer": true, "target_name": "count", "value_expression": "0" }
    private int count = 0;

    // 静态字段
    private static String TAG = "ORIGIN";

    public void testAssignments(int param) {
        // 1. 局部变量初始化 (Local Variable)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.testAssignments(int).local (VARIABLE)
        // Mores: { "is_initializer": true, "target_name": "local", "value_expression": "10", "raw_text": "int local = 10" }
        int local = 10;

        // 2. 本类字段赋值 (Implicit This)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.count (FIELD)
        // Mores: { "is_initializer": false, "target_name": "count", "operator": "+=", "value_expression": "5", "receiver": "this" }
        count += 5;

        // 3. 本类字段显式赋值 (Explicit This)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.count (FIELD)
        // Mores: { "is_initializer": false, "target_name": "count", "operator": "=", "value_expression": "100", "receiver": "this" }
        this.count = 100;

        // 4. 静态字段赋值 (Static Field)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.TAG (FIELD)
        // Mores: { "is_initializer": false, "target_name": "TAG", "operator": "=", "value_expression": "\"UPDATED\"", "receiver": "AssignRelationForClassSuite" }
        AssignRelationForClassSuite.TAG = "UPDATED";

        // 5. 跨对象字段赋值 (External Object Field)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.DataNode.name (FIELD)
        // Mores: { "is_initializer": false, "target_name": "name", "operator": "=", "value_expression": "\"NewName\"", "receiver": "node" }
        DataNode node = new DataNode();
        node.name = "NewName";

        // 6. 参数二次赋值 (Parameter)
        // Source: com.example.rel.AssignRelationForClassSuite.testAssignments(int) (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.testAssignments(int).param (VARIABLE)
        // Mores: { "is_initializer": false, "target_name": "param", "operator": "=", "value_expression": "200" }
        param = 200;
    }

    class DataNode {
        public String name;
    }
}