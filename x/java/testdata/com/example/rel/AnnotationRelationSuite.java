package com.example.rel;

import java.lang.annotation.*;

// 1. 类注解
// Source: com.example.rel.AnnotationRelationSuite
// Target: Entity
// Mores: { "java.rel.annotation.target": "TYPE" }
@Entity
// 1.1 类注解 - 带值
// Source: com.example.rel.AnnotationRelationSuite
// Target: SuppressWarnings
// Mores: { "java.rel.annotation.target": "TYPE", "java.rel.annotation.value": "\"all\"" }
@SuppressWarnings("all")
public class AnnotationRelationSuite {

    // 2. 字段注解
    // Source: com.example.rel.AnnotationRelationSuite.id
    // Target: Id
    // Mores: { "java.rel.annotation.target": "FIELD" }
    @Id
    @Column(name = "user_id", nullable = false)
    private Long id;

    // 3. 方法注解
    // Source: com.example.rel.AnnotationRelationSuite.save(String)
    // Target: Transactional
    // Mores: { "java.rel.annotation.target": "METHOD" }
    @Transactional(timeout = 100)
    public void save(@NotNull String data) {
        // 4. 局部变量注解
        // Source: com.example.rel.AnnotationRelationSuite.save(String).local
        // Target: NonEmpty
        // Mores: { "java.rel.annotation.target": "LOCAL_VARIABLE" }
        @NonEmpty String local = data;
    }
}