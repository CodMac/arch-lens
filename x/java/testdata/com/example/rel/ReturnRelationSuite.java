package com.example.rel;

import java.util.List;

public class ReturnRelationSuite {

    // 1. 对象返回
    // Source: com.example.rel.ReturnRelationSuite.getName
    // Target: java.lang.String
    // Mores: { "java.rel.return.is_primitive": false }
    public String getName() { return "test"; }

    // 2. 数组返回
    // Source: com.example.rel.ReturnRelationSuite.getBuffer
    // Target: byte
    // Mores: { "java.rel.return.is_array": true, "java.rel.return.dimensions": 1, "java.rel.return.is_primitive": true }
    public byte[] getBuffer() { return new byte[0]; }

    // 3. 泛型复合返回
    // Source: com.example.rel.ReturnRelationSuite.getValues
    // Target: java.util.List
    // Mores: { "java.rel.return.has_type_arguments": true }
    public List<Integer> getValues() { return null; }

    // 4. 基础类型返回
    // Source: com.example.rel.ReturnRelationSuite.getAge
    // Target: int
    // Mores: { "java.rel.return.is_primitive": true }
    public int getAge() { return 18; }

    // 5. 嵌套数组返回
    // Source: com.example.rel.ReturnRelationSuite.getMatrix
    // Target: double
    // Mores: { "java.rel.return.is_array": true, "java.rel.return.dimensions": 2 }
    public double[][] getMatrix() { return null; }
}