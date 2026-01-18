package com.example.rel;

import java.util.ArrayList;
import java.util.List;

public class AssignRelationForClassSuite {
    private Object globalObj;

    public void testClassAssignments() {
        // 1. 实例化并赋值 (Object Creation)
        // Source: Method(testClassAssignments), Target: Variable(list)
        // Expected Mores: { value_expression: "new ArrayList<>()", operator: "=" }
        List<String> list = new ArrayList<>();

        // 2. 将引用赋值给另一个变量 (Reference Copy)
        // Source: Method(testClassAssignments), Target: Variable(otherList)
        // Expected Mores: { value_expression: "list" }
        List<String> otherList = list;

        // 3. 跨对象字段赋值 (Field Access on Other Object)
        // Source: Method(testClassAssignments), Target: Field(name)
        // Expected Mores: { receiver: "data", value_expression: "\"Hello\"" }
        DataNode data = new DataNode();
        data.name = "Hello";

        // 4. 方法返回结果赋值
        // Source: Method(testClassAssignments), Target: Field(globalObj)
        // Expected Mores: { value_expression: "fetchObject()", receiver: "this" }
        this.globalObj = fetchObject();

        // 5. Null 赋值
        // Source: Method(testClassAssignments), Target: Variable(data)
        data = null;
    }

    private Object fetchObject() { return new Object(); }

    class DataNode {
        public String name;
    }
}