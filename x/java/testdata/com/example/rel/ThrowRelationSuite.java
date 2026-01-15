package com.example.rel;

import java.io.*;
import java.sql.SQLException;

public class ThrowRelationSuite {

    // 1. 方法签名中的声明 (Checked Exceptions)
    // Source: Method(readFile), Target: Class(IOException)
    // Mores: {
    //   "java.rel.throw.is_signature": true,
    //   "java.rel.throw.index": 0,
    //   "java.rel.ast_kind": "throws_clause"
    // }
    // Source: Method(readFile), Target: Class(SQLException)
    // Mores: {
    //   "java.rel.throw.is_signature": true,
    //   "java.rel.throw.index": 1,
    //   "java.rel.ast_kind": "throws_clause"
    // }
    public void readFile() throws IOException, SQLException {

        // 2. 方法体内主动抛出 (Throw Statement)
        // Source: Method(readFile), Target: Class(RuntimeException)
        // Mores: {
        //   "java.rel.throw.is_runtime": true,
        //   "java.rel.ast_kind": "throw_statement",
        //   "java.rel.raw_text": "throw new RuntimeException(\"Error\")"
        // }
        if (true) throw new RuntimeException("Error");
    }

    // 3. 构造函数声明抛出
    // Source: Constructor(ThrowRelationSuite), Target: Class(Exception)
    // Mores: {
    //   "java.rel.throw.is_signature": true,
    //   "java.rel.ast_kind": "throws_clause"
    // }
    public ThrowRelationSuite() throws Exception {}

    // 4. 重新抛出捕获的异常 (Rethrow)
    public void rethrow(Exception e) throws Exception {
        // Source: Method(rethrow), Target: Class(Exception)
        // Mores: { "java.rel.throw.is_rethrow": true, "java.rel.ast_kind": "throw_statement" }
        throw e;
    }
}