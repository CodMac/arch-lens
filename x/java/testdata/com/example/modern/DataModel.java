package com.example.modern;

/**
   * Record：参数即字段
   */
public record UserPoint(int x, int y) {
    // 允许有静态字段
    public static final String ORIGIN = "0,0";
}

/**
   * Sealed Class：限制继承
   */
public sealed interface Shape permits Circle, Square {}

final class Circle implements Shape {}
final class Square implements Shape {}