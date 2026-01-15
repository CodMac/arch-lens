package com.example.rel;

import java.lang.annotation.*;

// 1. 类注解
// Source: Class(AnnotationRelationSuite), Target: Class(Entity)
// Mores: { "java.rel.annotation_target": "TYPE" }
@Entity
@SuppressWarnings("all") // 带值的注解. Mores: { "java.rel.annotation_value": "\"all\"" }
public class AnnotationRelationSuite {

    // 2. 字段注解
    // Source: Field(id), Target: Class(Id)
    // Mores: { "java.rel.annotation_target": "FIELD" }
    @Id
    @Column(name = "user_id", nullable = false)
    // Mores: { "java.rel.annotation_params": "name=user_id,nullable=false" }
    private Long id;

    // 3. 方法与参数注解
    // Source: Method(save), Target: Class(Transactional)
    // Source: Parameter(data), Target: Class(NotNull)
    @Transactional(timeout = 100)
    public void save(@NotNull String data) {
        // 4. 局部变量注解 (部分框架支持)
        // Source: Variable(local), Target: Class(NonEmpty)
        @NonEmpty String local = data;
    }
}