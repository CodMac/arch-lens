package com.example.rel;

import java.io.Serializable;

// 1. 接口继承接口
interface BaseApi extends Serializable {
    void execute();
}

// 2. 标准单接口实现
interface SingleInterface {
    void run();
}

// 3. 多接口实现 + 泛型接口
class MultiImpl implements BaseApi, Runnable, SingleInterface {
    @Override public void execute() {}
    @Override public void run() {}
}

// 4. 抽象类实现接口
abstract class AbstractTask implements BaseApi {
    // 不实现 execute()，留给子类
}

public class ImplementRelationSuite {
    public void test() {
        // 5. 匿名内部类实现
        Runnable r = new Runnable() {
            @Override
            public void run() {
                System.out.println("Anonymous implement");
            }
        };
    }
}