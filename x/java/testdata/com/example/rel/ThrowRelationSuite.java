package com.example.rel;

import java.io.*;
import java.sql.SQLException;

public class ThrowRelationSuite {

    // 1. 方法签名中的声明
    // Source: com.example.rel.ThrowRelationSuite.readFile
    // Target: java.io.IOException
    // Mores: { "java.rel.throw.is_signature": true, "java.rel.throw.index": 0 }
    //
    // Source: com.example.rel.ThrowRelationSuite.readFile
    // Target: java.sql.SQLException
    // Mores: { "java.rel.throw.is_signature": true, "java.rel.throw.index": 1 }
    public void readFile() throws IOException, SQLException {

        // 2. 方法体内主动抛出
        // Source: com.example.rel.ThrowRelationSuite.readFile
        // Target: java.lang.RuntimeException
        // Mores: { "java.rel.raw_text": "throw new RuntimeException(\"Error\")" }
        if (true) throw new RuntimeException("Error");
    }

    // 3. 构造函数声明抛出
    // Source: com.example.rel.ThrowRelationSuite.<init>
    // Target: java.lang.Exception
    // Mores: { "java.rel.throw.is_signature": true }
    public ThrowRelationSuite() throws Exception {}

    // 4. 重新抛出捕获的异常
    // Source: com.example.rel.ThrowRelationSuite.rethrow
    // Target: java.lang.Exception
    public void rethrow() throws Exception {
        try {
            // ...
        } catch (Exception e) {
            throw e;
        }
    }
}