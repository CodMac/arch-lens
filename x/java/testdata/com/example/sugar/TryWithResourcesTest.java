package com.example.sugar;

import java.io.*;

public class TryWithResourcesTest {
  public void test() throws IOException {
    // 场景 1: 标准定义
    try (InputStream input = new FileInputStream("a.txt")) {
      input.read();
    }

    // 场景 2: 多个资源 (检查 QN 唯一性)
    try (OutputStream out = new FileOutputStream("b.txt");
      InputStream in = new FileInputStream("b.txt")) {
      out.flush();
    }
  }
}