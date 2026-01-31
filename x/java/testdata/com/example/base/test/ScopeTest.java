package com.example.base;

// 目标：验证 ScopeBlock 内的变量计数（$1, $2）以及 Lambda。
public class ScopeTest {
    public void test() {
        int x = 1;
        {
            int x = 2; // 验证 QN: ...test().block$1.x$1
        }
        if (true) {
            int x = 3; // 验证 QN: ...test().block$2.x$1
        }

        Runnable r = () -> {
            int x = 4; // Lambda 作用域

            int a, b, c = 5;
        };

        // 多参数无类型 (a, b) -> ...
        BinaryOperator<Integer> add = (p1, p2) -> {
            int sum = p1 + p2;
            return sum;
        };

        // 多参数带类型 (int x, int y) -> ...
        BinaryOperator<Integer> sub = (int v1, int v2) -> v1 - v2;

        // 单参数无括号 (之前已有类似，此处对比)
        java.util.function.Consumer<String> printer = s -> {
            String prefix = "LOG: ";
            System.out.println(prefix + s);
        };
    }
}