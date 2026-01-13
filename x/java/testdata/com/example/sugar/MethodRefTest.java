package com.example.sugar;

import java.util.function.*;
import java.util.List;
import java.util.ArrayList;

/**
 * 方法引用全覆盖测试案例
 */
public class MethodRefTest {

    public void testAllMethodReferences() {
        // 1. 静态方法引用 (Static Method Reference)
        // 预期 QN: ...testAllMethodReferences().method_ref$1, Signature: Integer::parseInt
        Function<String, Integer> staticRef = Integer::parseInt;

        // 2. 特定对象的实例方法引用 (Instance Method Reference on a Specific Object)
        // 预期 QN: ...testAllMethodReferences().method_ref$2, Signature: System.out::println
        Consumer<String> boundRef = System.out::println;

        // 3. 任意对象的实例方法引用 (Instance Method Reference on an Arbitrary Object of a particular type)
        // 预期 QN: ...testAllMethodReferences().method_ref$3, Signature: String::toLowerCase
        Function<String, String> arbitraryRef = String::toLowerCase;

        // 4. 构造函数引用 (Constructor Reference)
        // 预期 QN: ...testAllMethodReferences().method_ref$4, Signature: ArrayList::new
        Supplier<List<String>> constructorRef = ArrayList::new;

        // 5. 数组构造函数引用 (Array Constructor Reference)
        // 预期 QN: ...testAllMethodReferences().method_ref$5, Signature: int[]::new
        IntFunction<int[]> arrayRef = int[]::new;

        // 6. 泛型方法引用 (Generic Method Reference)
        // 预期 QN: ...testAllMethodReferences().method_ref$6, Signature: this::<String>genericMethod
        Consumer<String> genericRef = this::<String>genericMethod;
    }

    public <T> void genericMethod(T t) {
        System.out.println(t);
    }
}