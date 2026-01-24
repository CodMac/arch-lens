package com.example.rel;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CreateRelationSuite {

    // 1. 成员变量声明时实例化
    // Source: Field(com.example.rel.CreateRelationSuite.fieldInstance)
    // Target: Class(java.util.ArrayList)
    // Mores: { "java.rel.create.variable_name": "fieldInstance", "java.rel.ast_kind": "object_creation_expression" }
    private List<String> fieldInstance = new ArrayList<>();

    // 2. 静态成员变量实例化
    // Source: Field(com.example.rel.CreateRelationSuite.staticMap)
    // Target: Class(java.util.HashMap)
    // Mores: { "java.rel.create.variable_name": "staticMap", "java.rel.ast_kind": "object_creation_expression" }
    private static Map<String, String> staticMap = new HashMap<>();

    public void testCreateCases() {
        // 3. 局部变量实例化
        // Source: Method(com.example.rel.CreateRelationSuite.testCreateCases())
        // Target: Class(StringBuilder)
        // Mores: { "java.rel.create.variable_name": "sb", "java.rel.ast_kind": "object_creation_expression" }
        StringBuilder sb = new StringBuilder("init");

        // 4. 匿名内部类创建
        // Source: Method(com.example.rel.CreateRelationSuitetestCreateCases())
        // Target: Class(Runnable)
        // Mores: { "java.rel.create.variable_name": "r", "java.rel.ast_kind": "object_creation_expression" }
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running");
            }
        };

        // 5. 数组实例化
        // Source: Method(com.example.rel.CreateRelationSuitetestCreateCases())
        // Target: Class(String)
        // Mores: { "java.rel.create.is_array": true, "java.rel.ast_kind": "array_creation_expression" }
        String[] strings = new String[5];

        // 6. 链式调用中的实例化
        // Source: Method(com.example.rel.CreateRelationSuitetestCreateCases())
        // Target: Class(com.example.rel.CreateRelationSuite)
        // Mores: { "java.rel.ast_kind": "object_creation_expression" }
        new CreateRelationSuite().doNothing();
    }

    public CreateRelationSuite() {
        // 7. 构造函数内部实例化 (super)
        // Source: Constructor(com.example.rel.CreateRelationSuite.CreateRelationSuite())
        // Target: Class(Object)
        // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation" }
        super();
    }
}