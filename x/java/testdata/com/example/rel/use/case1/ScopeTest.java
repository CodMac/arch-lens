package com.example.rel.use.case1;

// 基础多层作用域
public class ScopeTest {
    private String name = "Field";

    public void test(String name) { // 参数遮蔽字段
        if (true) {
            String name = "Local"; // 局部变量遮蔽参数
            System.out.println(name); // [Case 1] 应解析为 Line 7 的 LocalVariable
        }
        System.out.println(name); // [Case 2] 应解析为 Line 5 的 Parameter
    }
}