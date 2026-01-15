package com.example.rel;

import java.util.Collection;
import java.util.List;

public class CastRelationSuite {

    public void testCastCases(Object input) {
        // 1. 基础对象向下转型 (Downcasting)
        // Source: Method(testCastCases), Target: Class(String)
        // Mores: {
        //   "java.rel.operand_expression": "input",
        //   "java.rel.operand_kind": "variable",
        //   "java.rel.raw_text": "(String) input"
        // }
        String s = (String) input;

        // 2. 基础数据类型转换 (Primitive Conversion)
        // Source: Method(testCastCases), Target: Type(int)
        // Mores: { "java.rel.operand_expression": "pi", "java.rel.cast_type": "primitive" }
        double pi = 3.14;
        int i = (int) pi;

        // 3. 泛型集合转型 (Generic Collection Cast)
        // Source: Method(testCastCases), Target: Class(List)
        // Mores: { "java.rel.type_arguments": "String", "java.rel.full_cast_text": "(List<String>)" }
        List<String> list = (List<String>) input;

        // 4. 链式调用中的转型 (Inline Cast)
        // Source: Method(testCastCases), Target: Class(SubClass)
        // Mores: { "java.rel.subsequent_call": "specificMethod", "java.rel.parent_ast": "parenthesized_expression" }
        ((SubClass) input).specificMethod();

        // 5. 模式匹配转型 (Pattern Matching for instanceof - Java 14+)
        // Source: Method(testCastCases), Target: Class(String)
        // Mores: {
        //   "java.rel.ast_kind": "instanceof_expression",
        //   "java.rel.is_pattern_matching": true,
        //   "java.rel.pattern_variable": "str"
        // }
        if (input instanceof String str) {
            System.out.println(str.length());
        }

        // 6. 多重转型 (Double Cast)
        // 产生两条 CAST 关系：
        // 1. Source: testCastCases -> Target: Object
        // 2. Source: testCastCases -> Target: Runnable
        // Mores: { "java.rel.is_nested_cast": true }
        ((Runnable)(Object)input).run();
    }

    static class SubClass {
        void specificMethod() {}
    }
}