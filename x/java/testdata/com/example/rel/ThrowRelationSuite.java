package com.example.rel;

import java.io.*;
import java.sql.SQLException;

public class ThrowRelationSuite {

    // 1. 单个与多个声明
    // Source: Method(readFile), Target: Class(IOException), Target: Class(SQLException)
    // Mores: { "java.rel.ast_kind": "throws_clause", "java.rel.is_signature": true }
    public void readFile() throws IOException, SQLException {

        // 2. 方法体内主动抛出 (既是 CREATE 也是 THROW 动作)
        // Source: Method(readFile), Target: Class(RuntimeException)
        // Mores: { "java.rel.ast_kind": "throw_statement", "java.rel.is_runtime": true }
        if (true) throw new RuntimeException("Error");
    }

    // 3. 构造函数声明
    // Source: Constructor(ThrowRelationSuite), Target: Class(Exception)
    public ThrowRelationSuite() throws Exception {}
}