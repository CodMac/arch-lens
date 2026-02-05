package com.example.rel.use.case5;

public class ClosureTest {
    private String context = "Outer";

    public void run() {
        Runnable r = new Runnable() {
            private String context = "Inner";
            @Override
            public void run() {
                System.out.println(context); // [Case 9] 应解析为匿名内部类的 Field
            }
        };

        list.forEach(item -> {
            System.out.println(context); // [Case 10] 应解析为外部类的 Field (Lambda 闭包)
        });
    }
}