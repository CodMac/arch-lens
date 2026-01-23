package com.example.rel;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CreateRelationSuite {

    // 1. 成员变量声明时实例化 (Field Initializer)
    // Source: Field(com.example.rel.CreateRelationSuite.fieldInstance)
    // Target: Class(java.util.ArrayList)
    // Mores: { "java.rel.assign.is_initializer": true, "java.rel.create.variable_name": "fieldInstance", "java.rel.ast_kind": "object_creation_expression" }
    private List<String> fieldInstance = new ArrayList<>();

    // 2. 静态成员变量实例化 (Static Field Initializer)
    // Source: Field(com.example.rel.CreateRelationSuite.staticMap)
    // Target: Class(java.util.HashMap)
    // Mores: { "java.rel.call.is_static": true, "java.rel.assign.is_initializer": true, "java.rel.ast_kind": "object_creation_expression" }
    private static Map<String, String> staticMap = new HashMap<>();

    public void testCreateCases() {
        // 3. 局部变量实例化 (Local Variable Creation)
        // Source: Method(com.example.rel.CreateRelationSuite.testCreateCases())
        // Target: Class(java.lang.StringBuilder)
        // Mores: { "java.rel.create.variable_name": "sb", "java.rel.ast_kind": "object_creation_expression" }
        StringBuilder sb = new StringBuilder("init");

        // 4. 匿名内部类创建 (Anonymous Class Creation)
        // Source: Method(testCreateCases), Target: Interface(java.lang.Runnable)
        // Mores: { "java.rel.ast_kind": "object_creation_expression", "java.rel.call.is_constructor": true }
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running");
            }
        };

        // 5. 数组实例化 (Array Creation)
        // Source: Method(testCreateCases), Target: Class(java.lang.String)
        // Mores: { "java.rel.create.is_array": true, "java.rel.ast_kind": "array_creation_expression" }
        String[] strings = new String[5];

        // 6. 链式调用中的实例化 (In-chain Creation)
        // Source: Method(testCreateCases), Target: Class(com.example.rel.CreateRelationSuite)
        // Mores: { "java.rel.call.is_chained": true, "java.rel.ast_kind": "object_creation_expression" }
        new CreateRelationSuite().doNothing();
    }

    public CreateRelationSuite() {
        // 7. 构造函数内部实例化 (显式父类构造调用)
        // Source: Constructor(com.example.rel.CreateRelationSuite.CreateRelationSuite())
        // Target: Class(java.lang.Object)
        // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.call.receiver": "super", "java.rel.call.is_constructor": true }
        super();
    }

    public void doNothing() {}
}