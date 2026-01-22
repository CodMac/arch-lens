package com.example.rel;

public class AssignRelationForDataFlow {
    private String data;

    public void testDataFlow() {
        // 1. 常量赋值 (Constant Flow)
        // Source: com.example.rel.AssignRelationForDataFlow.testDataFlow() (METHOD)
        // Target: com.example.rel.AssignRelationForDataFlow.data (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "this.data = \"CONST\"",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.assign.target_name": "data",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "\"CONST\""
        // }
        this.data = "CONST";

        // 2. 返回值流向 (Return Value Flow)
        // Source: com.example.rel.AssignRelationForDataFlow.testDataFlow() (METHOD)
        // Target: com.example.rel.AssignRelationForDataFlow.testDataFlow().localObj (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "localObj = fetch()",
        //   "java.rel.ast_kind": "variable_declarator",
        //   "java.rel.assign.target_name": "localObj",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "fetch()"
        // }
        Object localObj = fetch();

        // 3. 转换流向 (Cast Flow)
        // Source: com.example.rel.AssignRelationForDataFlow.testDataFlow() (METHOD)
        // Target: com.example.rel.AssignRelationForDataFlow.testDataFlow().msg (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "msg = (String) localObj",
        //   "java.rel.ast_kind": "variable_declarator",
        //   "java.rel.assign.target_name": "msg",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "(String) localObj"
        // }
        String msg = (String) localObj;
    }

    private Object fetch() { return "hello"; }
}