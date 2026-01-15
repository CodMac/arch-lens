package com.example.rel;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CreateRelationSuite {

    // 1. 成员变量声明时实例化 (Field Initializer)
    // Source: Field(fieldInstance), Target: Class(ArrayList)
    // Mores: {
    //   "java.rel.create.is_initializer": true,
    //   "java.rel.create.variable_name": "fieldInstance",
    //   "java.rel.ast_kind": "object_creation_expression",
    //   "java.rel.raw_text": "new ArrayList<>()"
    // }
    private List<String> fieldInstance = new ArrayList<>();

    // 2. 静态成员变量实例化 (Static Field Initializer)
    // Source: Field(staticMap), Target: Class(HashMap)
    // Mores: {
    //   "java.rel.call.is_static": true,
    //   "java.rel.create.is_initializer": true,
    //   "java.rel.ast_kind": "object_creation_expression"
    // }
    private static Map<String, String> staticMap = new HashMap<>();

    public void testCreateCases() {
        // 3. 局部变量实例化 (Local Variable Creation)
        // Source: Method(testCreateCases), Target: Class(StringBuilder)
        // Mores: {
        //   "java.rel.create.variable_name": "sb",
        //   "java.rel.create.arguments": "\"init\"",
        //   "java.rel.ast_kind": "object_creation_expression"
        // }
        StringBuilder sb = new StringBuilder("init");

        // 4. 匿名内部类创建 (Anonymous Class Creation)
        // Source: Method(testCreateCases), Target: Interface(Runnable)
        // Mores: {
        //   "java.rel.create.is_anonymous": true,
        //   "java.rel.ast_kind": "anonymous_class_submission"
        // }
        // 注意：这会同时触发对 Runnable 的 IMPLEMENT 关系
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("Running");
            }
        };

        // 5. 数组实例化 (Array Creation)
        // Source: Method(testCreateCases), Target: Class(String)
        // Mores: {
        //   "java.rel.create.is_array": true,
        //   "java.rel.create.dimensions": 1,
        //   "java.rel.create.array_size": "5",
        //   "java.rel.ast_kind": "array_creation_expression"
        // }
        String[] strings = new String[5];

        // 6. 链式调用中的实例化 (In-chain Creation)
        // Source: Method(testCreateCases), Target: Class(CreateRelationSuite)
        // Mores: {
        //   "java.rel.create.has_subsequent_call": true,
        //   "java.rel.create.subsequent_call": "doNothing",
        //   "java.rel.ast_kind": "object_creation_expression"
        // }
        new CreateRelationSuite().doNothing();
    }

    public CreateRelationSuite() {
        // 7. 构造函数内部实例化 (显式父类构造调用)
        // Source: Constructor(CreateRelationSuite), Target: Class(Object)
        // Mores: {
        //   "java.rel.ast_kind": "explicit_constructor_invocation",
        //   "java.rel.call.receiver": "super",
        //   "java.rel.create.is_constructor_chain": true
        // }
        super();
    }

    public void doNothing() {}
}