package com.example.rel;

import java.lang.annotation.Retention;

public class ParameterRelationSuite {

    // 4. 构造函数参数
    // Source: com.example.rel.ParameterRelationSuite.<init> (或 ParameterRelationSuite)
    // Target: int
    // Mores: { "java.rel.parameter.name": "val", "java.rel.parameter.index": 0 }
    public ParameterRelationSuite(int val) {}

    // 1. 多参数顺序与类型
    // Source: com.example.rel.ParameterRelationSuite.update
    // Target: String (Index: 0), long (Index: 1)
    // Mores: { "java.rel.parameter.name": "name", "java.rel.parameter.index": 0 }
    public void update(String name, long id) {}

    // 2. 可变参数 (Varargs)
    // Source: com.example.rel.ParameterRelationSuite.log
    // Target: Object
    // Mores: { "java.rel.parameter.is_varargs": true, "java.rel.parameter.index": 1 }
    public void log(String message, Object... args) {}

    // 3. Final 参数与注解修饰
    // Source: com.example.rel.ParameterRelationSuite.setPath
    // Target: String
    // Mores: { "java.rel.parameter.is_final": true, "java.rel.parameter.has_annotation": true }
    public void setPath(@Deprecated final String path) {}
}