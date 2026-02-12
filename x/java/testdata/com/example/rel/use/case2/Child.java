package com.example.rel.use.case2;

public class Child extends Parent {
    private int count = 20; // 遮蔽父类字段

    public void print() {
        System.out.println(count); // [Case 3] 应解析为本类的 Field (count=20)
        System.out.println(TAG);   // [Case 4] 应跨类解析为 Parent 的 Static Field
    }
}