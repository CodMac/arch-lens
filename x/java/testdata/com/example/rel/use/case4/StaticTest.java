package com.example.rel.use.case4;

public class StaticTest {
    private int instanceVar = 1;
    private static int staticVar = 2;

    public static void staticMethod() {
        System.out.println(staticVar);   // [Case 7] 应解析成功
        System.out.println(instanceVar); // [Case 8] 应解析失败 (静态方法不能引用非静态变量)
    }
}