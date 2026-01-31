package com.example.rel;

import java.util.ArrayList;
import java.io.Serializable;

// 1. 类继承 (带泛型擦除)
@Deprecated
public class ExtendRelationSuite extends ArrayList<String> {

    // 2. 接口多继承
    interface SubInterface extends Runnable, Serializable {}

    public void test() {
        // 3. 匿名内部类继承
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("running");
            }
        };
    }
}