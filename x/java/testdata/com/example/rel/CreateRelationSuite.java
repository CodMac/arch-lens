package com.example.rel;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CreateRelationSuite {

    // 1. 成员变量声明时实例化 (Field Initializer)
    // Source: Field(fieldInstance), Target: Class(ArrayList)
    // Mores: {
    //   "java.rel.is_initializer": true,
    //   "java.rel.variable_name": "fieldInstance",
    //   "java.rel.raw_text": "new ArrayList<>()"
    // }
    private List<String> fieldInstance = new ArrayList<>();

    // 2. 静态成员变量实例化 (Static Field Initializer)
    // Source: Field(staticMap), Target: Class(HashMap)
    // Mores: { "java.rel.is_static": true, "java.rel.is_initializer": true }
    private static Map<String, String> staticMap = new HashMap<>();

    public void testCreateCases() {
        // 3. 局部变量实例化 (Local Variable Creation)
        // Source: Method(testCreateCases), Target: Class(StringBuilder)
        // Mores: { "java.rel.variable_name": "sb", "java.rel.arguments": "\"init\"" }
        StringBuilder sb = new StringBuilder("init");

        // 4. 匿名内部类创建 (Anonymous Class Creation)
        // Source: Method(testCreateCases), Target: Interface(Runnable)
        // Mores: { "java.rel.is_anonymous": true, "java.rel.ast_kind": "object_creation_expression" }
        // 注意：这会同时触发对 Runnable 的 IMPLEMENT 关系
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running");
            }
        };

        // 5. 数组实例化 (Array Creation)
        // Source: Method(testCreateCases), Target: Class(String)
        // Mores: { "java.rel.is_array": true, "java.rel.dimensions": 1, "java.rel.array_size": "5" }
        String[] strings = new String[5];

        // 6. 链式调用中的实例化 (In-chain Creation)
        // Source: Method(testCreateCases), Target: Class(CreateRelationSuite)
        // Mores: { "java.rel.has_subsequent_call": true, "java.rel.subsequent_call": "doNothing" }
        new CreateRelationSuite().doNothing();
    }

    public CreateRelationSuite() {
        // 7. 构造函数内部实例化
        // Source: Constructor(CreateRelationSuite), Target: Class(Object)
        // Mores: { "java.rel.ast_kind": "explicit_constructor_invocation", "java.rel.receiver": "super" }
        super();
    }

    public void doNothing() {}
}