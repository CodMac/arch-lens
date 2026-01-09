package com.example.base.test;

import java.io.Serializable;

public class ExtendAndImplementTest {}

// 目标：验证 IsAbstract, IsFinal, SuperClass, Annotations。
@Deprecated
@SuppressWarnings("unused")
abstract class BaseClass implements Serializable {
}

final class FinalClass extends BaseClass implements Cloneable, Runnable {
    @Override public void run() {}
}