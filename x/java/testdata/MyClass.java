package com.example.app;

import java.util.List;

@MyClassAnnotation1("MyClass")
@MyClassAnnotation2(value = "MyClass")
@MyClassAnnotation3
public class MyClass implements MyInterface {

    // Field 1
    private final String APP_NAME = "MyApp";

    // Field 2
    @MyFieldAnnotation("counter")
    public int counter = 0;

    // Constructor
    public MyClass(String name) {
        this.name = name;
    }

    // Method
    public List<String> run(int times) {
        return null;
    }

    public static class MyInnerClass {
        private String a;
    }
}

class InnerClass {
    private String a;
    private String b = "b";
}

interface MyInterface {
    void process();
}

enum Status {
    ACTIVE, // Enum Constant
    INACTIVE(0) // Enum Constant with arguments
}

@interface MyClassAnnotation1 { String value(); }
@interface MyClassAnnotation2 { String value(); }
@interface MyClassAnnotation3 {}
@interface MyFieldAnnotation { String value(); }