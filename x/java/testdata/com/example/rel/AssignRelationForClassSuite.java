package com.example.rel;

import java.util.ArrayList;
import java.util.List;

public class AssignRelationForClassSuite {
    private Object globalObj;

    public void testClassAssignments() {
        // 1. 实例化并赋值 (Object Creation)
        // Source: com.example.rel.AssignRelationForClassSuite.testClassAssignments() (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.testClassAssignments().list (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "list = new ArrayList<>()",
        //   "java.rel.ast_kind": "variable_declarator",
        //   "java.rel.context": "variable_declarator",
        //   "java.rel.assign.target_name": "list",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "new ArrayList<>()"
        // }
        List<String> list = new ArrayList<>();

        // 2. 将引用赋值给另一个变量 (Reference Copy)
        // Source: com.example.rel.AssignRelationForClassSuite.testClassAssignments() (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.testClassAssignments().otherList (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "otherList = list",
        //   "java.rel.ast_kind": "variable_declarator",
        //   "java.rel.context": "variable_declarator",
        //   "java.rel.assign.target_name": "otherList",
        //   "java.rel.assign.is_initializer": true,
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "list"
        // }
        List<String> otherList = list;

        // 3. 跨对象字段赋值 (Field Access)
        // Source: com.example.rel.AssignRelationForClassSuite.testClassAssignments() (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.DataNode.name (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "data.name = \"Hello\"",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.context": "expression_statement",
        //   "java.rel.assign.target_name": "name",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "\"Hello\""
        // }
        DataNode data = new DataNode();
        data.name = "Hello";

        // 4. 方法返回结果赋值
        // Source: com.example.rel.AssignRelationForClassSuite.testClassAssignments() (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.globalObj (FIELD)
        // Mores: {
        //   "java.rel.raw_text": "this.globalObj = fetchObject()",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.context": "expression_statement",
        //   "java.rel.assign.target_name": "globalObj",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "fetchObject()"
        // }
        this.globalObj = fetchObject();

        // 5. Null 赋值
        // Source: com.example.rel.AssignRelationForClassSuite.testClassAssignments() (METHOD)
        // Target: com.example.rel.AssignRelationForClassSuite.testClassAssignments().data (VARIABLE)
        // Mores: {
        //   "java.rel.raw_text": "data = null",
        //   "java.rel.ast_kind": "assignment_expression",
        //   "java.rel.context": "expression_statement",
        //   "java.rel.assign.target_name": "data",
        //   "java.rel.assign.operator": "=",
        //   "java.rel.assign.value_expression": "null"
        // }
        data = null;
    }

    private Object fetchObject() { return new Object(); }

    class DataNode {
        public String name;
    }
}