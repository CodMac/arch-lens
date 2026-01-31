package com.example.sugar;

/**
* 挑战 Record：一个声明，多个定义
*/
public record User(Long id, String name) {
  // 显式定义的方法，用于验证不会重复生成
  public String name() {
    return name.toUpperCase();
  }
}