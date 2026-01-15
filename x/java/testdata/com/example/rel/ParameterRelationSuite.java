package com.example.rel;

import java.lang.annotation.Retention;

public class ParameterRelationSuite {

    // 1. 多参数顺序与类型
    // Source: Method(update), Target: Class(String)
    // Mores: {
    //   "java.rel.parameter.name": "name",
    //   "java.rel.parameter.index": 0,
    //   "java.rel.ast_kind": "formal_parameter"
    // }
    // Source: Method(update), Target: Type(long)
    // Mores: {
    //   "java.rel.parameter.name": "id",
    //   "java.rel.parameter.index": 1,
    //   "java.rel.ast_kind": "formal_parameter"
    // }
    public void update(String name, long id) {}

    // 2. 可变参数 (Varargs)
    // Source: Method(log), Target: Class(Object)
    // Mores: {
    //   "java.rel.parameter.is_varargs": true,
    //   "java.rel.parameter.name": "args",
    //   "java.rel.parameter.index": 1,
    //   "java.rel.ast_kind": "spread_parameter"
    // }
    public void log(String message, Object... args) {}

    // 3. Final 参数与注解修饰
    // Source: Method(setPath), Target: Class(String)
    // Mores: {
    //   "java.rel.parameter.is_final": true,
    //   "java.rel.parameter.name": "path",
    //   "java.rel.parameter.index": 0,
    //   "java.rel.parameter.has_annotation": true
    // }
    public void setPath(@Deprecated final String path) {}

    // 4. 构造函数参数 (特殊的 Parameter 场景)
    // Source: Constructor(ParameterRelationSuite), Target: Type(int)
    // Mores: { "java.rel.parameter.index": 0, "java.rel.parameter.name": "val" }
    public ParameterRelationSuite(int val) {}
}