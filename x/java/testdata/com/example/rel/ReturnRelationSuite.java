package com.example.rel;

import java.util.List;

public class ReturnRelationSuite {

    // 1. 对象返回
    // Source: Method(getName), Target: Class(String)
    public String getName() { return "test"; }

    // 2. 数组返回
    // Source: Method(getBuffer), Target: Type(byte)
    // Mores: { "java.rel.is_array": true, "java.rel.dimensions": 1 }
    public byte[] getBuffer() { return new byte[0]; }

    // 3. 泛型复合返回
    // Source: Method(getValues), Target: Class(List)
    // 配合 TypeArg 提取：Target: Class(Integer)
    public List<Integer> getValues() { return null; }

    // 4. 基础类型 (通常排除 void，但保留 int/double 等)
    // Source: Method(getAge), Target: Type(int)
    public int getAge() { return 18; }
}