package com.example.rel;

public class AssignRelationSuiteForDataFlow {
    private String data;

    public void testDataFlow() {
        // 1. 常量赋值 (Constant Flow)
        // Target: ...data (FIELD)
        // Mores: {
        //   "java.rel.assign.is_constant": true,
        //   "java.rel.assign.value_expression": "\"CONST\""
        // }
        this.data = "CONST";

        // 2. 返回值流向 (Return Value Flow)
        // Target: ...localObj (VARIABLE)
        // Mores: {
        //   "java.rel.assign.is_return_value": true,
        //   "java.rel.assign.value_expression": "fetch()"
        // }
        Object localObj = fetch();

        // 3. 转换流向 (Cast Flow)
        // Target: ...msg (VARIABLE)
        // Mores: {
        //   "java.rel.assign.is_cast_check": true,
        //   "java.rel.assign.cast_type": "String",
        //   "java.rel.assign.value_expression": "localObj"
        // }
        String msg = (String) localObj;
    }

    private Object fetch() { return "hello"; }
}