package com.example.app;

import java.util.List;

@MyClassAnnotation("MyClass")
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
}

interface MyInterface {
    void process();
}

enum Status {
    ACTIVE, // Enum Constant
    INACTIVE(0) // Enum Constant with arguments
}

@interface MyClassAnnotation { String value(); }
@interface MyFieldAnnotation { String value(); }