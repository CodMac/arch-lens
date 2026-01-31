package com.example.base.test;

import java.util.List;

public class ScopeVariableTest {
    public void test() {
        // 1. try-with-resources
        try (BufferedInputStream bis = new BufferedInputStream(null)) {
            bis.read();
        } catch (Exception e) { // catch_clause
            e.printStackTrace();
        }

        // 2. traditional for loop
        for (int i = 0; i < 10; i++) {
            System.out.println(i);
        }

        // 3. enhanced for loop
        List<String> list = null;
        for (String item : list) {
            System.out.println(item);
        }

        // 4. Pattern Matching for instanceof (Java 14+)
        Object obj = "hello";
        if (obj instanceof String s) {
            System.out.println(s);
        }
    }
}