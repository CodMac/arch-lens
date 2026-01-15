package com.example.rel;

import java.util.*;
import java.io.Serializable;

public class TypeArgRelationSuite {

    // 1. 基础多泛型
    // Source: Field(map), Target: Class(String)
    // Mores: {
    //   "java.rel.type_arg.parent_type": "Map",
    //   "java.rel.type_arg.index": 0,
    //   "java.rel.ast_kind": "type_arguments"
    // }
    // Source: Field(map), Target: Class(Integer)
    // Mores: { "java.rel.type_arg.parent_type": "Map", "java.rel.type_arg.index": 1 }
    private Map<String, Integer> map;

    // 2. 嵌套泛型 (Nested Generics)
    // Source: Field(complexList), Target: Class(Map)
    // Mores: {
    //   "java.rel.type_arg.depth": 1,
    //   "java.rel.type_arg.parent_type": "List",
    //   "java.rel.type_arg.index": 0
    // }
    // Source: Field(complexList), Target: Class(Object)
    // Mores: {
    //   "java.rel.type_arg.depth": 2,
    //   "java.rel.type_arg.parent_type": "Map",
    //   "java.rel.type_arg.index": 1
    // }
    private List<Map<String, Object>> complexList;

    // 3. 通配符与边界 (Wildcards & Bounds)
    // Source: Method(process), Target: Class(Serializable)
    // Mores: {
    //   "java.rel.type_arg.wildcard_kind": "extends",
    //   "java.rel.type_arg.is_wildcard": true,
    //   "java.rel.raw_text": "? extends Serializable"
    // }
    public void process(List<? extends Serializable> input) {

        // 4. 构造函数泛型实参 (Explicit Type Arguments in Creation)
        // Source: Variable(list), Target: Class(String)
        // Mores: {
        //   "java.rel.type_arg.parent_type": "ArrayList",
        //   "java.rel.ast_kind": "type_arguments"
        // }
        List<String> list = new ArrayList<String>();
    }

    // 5. 下界通配符 (Lower Bounds)
    // Source: Method(addNumbers), Target: Class(Integer)
    // Mores: { "java.rel.type_arg.wildcard_kind": "super", "java.rel.type_arg.is_wildcard": true }
    public void addNumbers(List<? super Integer> list) {}
}