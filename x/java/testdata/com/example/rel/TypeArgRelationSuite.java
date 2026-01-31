package com.example.rel;

import java.util.*;
import java.io.Serializable;

public class TypeArgRelationSuite {

    // 1. 基础多泛型
    // Source: com.example.rel.TypeArgRelationSuite.map, Target: String
    // Mores: { "java.rel.type_arg.index": 0, "java.rel.ast_kind": "type_arguments" }
    // Source: com.example.rel.TypeArgRelationSuite.map, Target: Integer
    // Mores: { "java.rel.type_arg.index": 1 }
    private Map<String, Integer> map;

    // 2. 嵌套泛型
    // Source: com.example.rel.TypeArgRelationSuite.complexList, Target: java.util.Map
    // Mores: { "java.rel.type_arg.index": 0 }
    // Source: com.example.rel.TypeArgRelationSuite.complexList, Target: Object
    // Mores: { "java.rel.type_arg.index": 1 } // 这里指 Map 内部的第二个参数 Object
    private List<Map<String, Object>> complexList;

    // 3. 通配符与上界
    // Source: com.example.rel.TypeArgRelationSuite.process.input, Target: java.io.Serializable
    // Mores: { "java.rel.type_arg.index": 0, "java.rel.raw_text": "? extends Serializable" }
    public void process(List<? extends Serializable> input) {

        // 4. 构造函数泛型实参
        // Source: com.example.rel.TypeArgRelationSuite.process.list, Target: String
        // Mores: { "java.rel.type_arg.index": 0, "java.rel.ast_kind": "type_arguments" }
        List<String> list = new ArrayList<String>();
    }

    // 5. 下界通配符
    // Source: com.example.rel.TypeArgRelationSuite.addNumbers.list, Target: Integer
    // Mores: { "java.rel.type_arg.index": 0, "java.rel.raw_text": "? super Integer" }
    public void addNumbers(List<? super Integer> list) {}
}