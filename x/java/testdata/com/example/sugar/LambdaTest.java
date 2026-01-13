package com.example.sugar;

import java.util.function.*;

public class LambdaTest {
  public void testLambda() {
    // 1. 隐式类型参数 (Inferred types)
    BiFunction<Integer, Integer, Integer> adder = (a, b) -> a + b;

    // 2. 单参数括号省略 (Implicit single param)
    Consumer<String> printer = s -> {
      // 3. Lambda 内部的局部变量
      String prefix = "LOG: ";
      System.out.println(prefix + s);
    };

    // 4. 显式类型参数 (Explicit types)
    BinaryOperator<Long> multiplier = (Long x, Long y) -> x * y;
  }
}