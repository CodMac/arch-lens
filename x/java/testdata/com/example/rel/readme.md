#### 关系提取的测试用例

##### 基础测试用例
- 位置: `x/java/testdata/com/example/rel`
    ```
      x/java/testdata/com/example/rel/TypeArgRelationSuite.java
      x/java/testdata/com/example/rel/ParameterRelationSuite.java
      x/java/testdata/com/example/rel/AssignRelationSuite.java
      x/java/testdata/com/example/rel/CallRelationSuite.java
      x/java/testdata/com/example/rel/UseRelationSuite.java
      x/java/testdata/com/example/rel/AssignRelationForClassSuite.java
      x/java/testdata/com/example/rel/ThrowRelationSuite.java
      x/java/testdata/com/example/rel/ImplementRelationSuite.java
      x/java/testdata/com/example/rel/AssignRelationForDataFlow.java
      x/java/testdata/com/example/rel/CastRelationSuite.java
      x/java/testdata/com/example/rel/ReturnRelationSuite.java
      x/java/testdata/com/example/rel/AnnotationRelationSuite.java
      x/java/testdata/com/example/rel/CreateRelationSuite.java
      x/java/testdata/com/example/rel/ExtendRelationSuite.java
      x/java/testdata/com/example/rel/CaptureRelationSuite.java
    ```

##### 进阶测试用例
- Use关系:
    ```
    # Case1: 基础多层作用域 (Lexical Scoping)
    x/java/testdata/com/example/rel/use/ScopeTest.java
    
    # Case2: 类成员与继承 (Inheritance & Shadowing)
    x/java/testdata/com/example/rel/use/Parent.java
    x/java/testdata/com/example/rel/use/Child.java
    
    # Case3: 跨包可见性 (Cross-Package Visibility)
    x/java/testdata/com/example/rel/use/case3/Base.java
    x/java/testdata/com/example/rel/use/case3/Sub.java
    
    # Case4: 静态上下文约束 (Static Context)
    x/java/testdata/com/example/rel/use/case4/StaticTest.java
    
    # Case5: 匿名内部类与 Lambda (Closure)
    x/java/testdata/com/example/rel/use/case5/ClosureTest.java
    
    # Case6: 流式调用中的 Receiver 溯源
    x/java/testdata/com/example/rel/use/case6/ReceiverTest.java
    ```
