package com.example.rel;

import java.util.List;

public class ReturnRelationSuite {

    // 1. 对象返回
    // Source: Method(getName), Target: Class(String)
    // Mores: {
    //   "java.rel.return.is_primitive": false,
    //   "java.rel.ast_kind": "method_declaration"
    // }
    public String getName() { return "test"; }

    // 2. 数组返回
    // Source: Method(getBuffer), Target: Type(byte)
    // Mores: {
    //   "java.rel.return.is_array": true,
    //   "java.rel.return.dimensions": 1,
    //   "java.rel.return.is_primitive": true
    // }
    public byte[] getBuffer() { return new byte[0]; }

    // 3. 泛型复合返回
    // Source: Method(getValues), Target: Class(List)
    // Mores: {
    //   "java.rel.return.has_type_arguments": true,
    //   "java.rel.ast_kind": "generic_type"
    // }
    // 注意：List<Integer> 还会触发 TypeArg 关系：[Method(getValues) -> Class(Integer)]
    public List<Integer> getValues() { return null; }

    // 4. 基础类型返回
    // Source: Method(getAge), Target: Type(int)
    // Mores: { "java.rel.return.is_primitive": true }
    public int getAge() { return 18; }

    // 5. 嵌套数组返回 (深度测试)
    // Source: Method(getMatrix), Target: Type(double)
    // Mores: { "java.rel.return.is_array": true, "java.rel.return.dimensions": 2 }
    public double[][] getMatrix() { return new double[0][0]; }
}