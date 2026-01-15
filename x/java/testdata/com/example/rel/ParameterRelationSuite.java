package com.example.rel;

public class ParameterRelationSuite {

    // 1. 多参数顺序
    // Source: Method(update), Target: Class(String), Target: Type(long)
    // Mores: { "java.rel.param_name": "name", "java.rel.index": 0 }
    // Mores: { "java.rel.param_name": "id", "java.rel.index": 1 }
    public void update(String name, long id) {}

    // 2. 可变参数 (Varargs)
    // Source: Method(log), Target: Class(Object)
    // Mores: { "java.rel.is_varargs": true, "java.rel.param_name": "args" }
    public void log(String message, Object... args) {}

    // 3. Final 参数
    // Source: Method(setPath), Target: Class(String)
    // Mores: { "java.rel.is_final": true }
    public void setPath(final String path) {}
}