package com.example.rel;

import java.util.Collection;
import java.util.List;

public class CastRelationSuite {

    public void testCastCases(Object input) {
        // 1. 基础对象向下转型 (Downcasting)
        // Source: Method(com.example.rel.CastRelationSuite.testCastCases(Object))
        // Target: Class(java.lang.String)
        // Mores: { "java.rel.ast_kind": "cast_expression", "java.rel.raw_text": "(String) input" }
        String s = (String) input;

        // 2. 基础数据类型转换 (Primitive Conversion)
        // Source: Method(testCastCases), Target: Class(int)
        // Mores: { "java.rel.ast_kind": "cast_expression" }
        double pi = 3.14;
        int i = (int) pi;

        // 3. 泛型集合转型 (Generic Collection Cast)
        // Source: Method(testCastCases), Target: Class(java.util.List)
        // Mores: { "java.rel.ast_kind": "cast_expression" }
        List<String> list = (List<String>) input;

        // 4. 链式调用中的转型 (Inline Cast)
        // Source: Method(testCastCases), Target: Class(com.example.rel.CastRelationSuite.SubClass)
        // Mores: { "java.rel.ast_kind": "cast_expression" }
        ((SubClass) input).specificMethod();

        // 5. 模式匹配转型 (Pattern Matching for instanceof - Java 14+)
        // Source: Method(testCastCases), Target: Class(java.lang.String)
        // Mores: { "java.rel.ast_kind": "instanceof_expression", "java.rel.raw_text": "input instanceof String str" }
        if (input instanceof String str) {
            System.out.println(str.length());
        }

        // 6. 多重转型 (Double Cast)
        // Source: Method(testCastCases), Target1: Class(java.lang.Object), Target2: Class(java.lang.Runnable)
        // Mores: { "java.rel.ast_kind": "cast_expression" }
        ((Runnable)(Object)input).run();
    }

    static class SubClass {
        void specificMethod() {}
    }
}