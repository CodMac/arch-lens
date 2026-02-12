package com.example.rel.use.case6;

public class ReceiverTest {
    public void test() {
        User user = new User();
        user.getName().trim(); // [Case 11] 解析 getName() 时，能否识别出 user 是 User 类型？
    }
}