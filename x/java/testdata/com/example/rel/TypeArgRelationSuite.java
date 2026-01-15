package com.example.rel;

import java.util.*;
import java.io.Serializable;

public class TypeArgRelationSuite {

    // 1. 基础多泛型
    // Source: Field(map), Target: Class(String), Target: Class(Integer)
    // Mores: { "java.rel.container": "Map", "java.rel.arg_index": 0 }
    private Map<String, Integer> map;

    // 2. 嵌套泛型 (这是 Extractor 最容易出错的地方)
    // Source: Field(complexList), Target: Class(List), Target: Class(Map), Target: Class(String), Target: Class(Object)
    // Mores: { "java.rel.depth": 2, "java.rel.parent_arg": "Map" }
    private List<Map<String, Object>> complexList;

    // 3. 通配符与边界 (Wildcards)
    // Source: Method(process), Target: Class(Serializable)
    // Mores: { "java.rel.wildcard": "extends", "java.rel.raw_text": "? extends Serializable" }
    public void process(List<? extends Serializable> input) {
        // 4. 构造函数泛型实参
        // Source: Method(process), Target: Class(String)
        List<String> list = new ArrayList<String>();
    }
}