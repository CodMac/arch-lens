package com.example.rel;

import java.lang.annotation.*;

// 1. 类注解 - 无参 (Marker Annotation)
// Source: Class(AnnotationRelationSuite), Target: Class(Entity)
// Mores: {
//   "java.rel.annotation.target": "TYPE",
//   "java.rel.raw_text": "@Entity"
// }
@Entity
// 1.1 类注解 - 单参数 (Single Element Annotation)
// Source: Class(AnnotationRelationSuite), Target: Class(SuppressWarnings)
// Mores: {
//   "java.rel.annotation.target": "TYPE",
//   "java.rel.annotation.value": "\"all\"",
//   "java.rel.raw_text": "@SuppressWarnings(\"all\")"
// }
@SuppressWarnings("all")
public class AnnotationRelationSuite {

    // 2. 字段注解 - 无参
    // Source: Field(id), Target: Class(Id)
    // Mores: {
    //   "java.rel.annotation.target": "FIELD",
    // }
    @Id
    // 2.1 字段注解 - 多参数 (Normal Annotation)
    // Source: Field(id), Target: Class(Column)
    // Mores: {
    //   "java.rel.annotation.target": "FIELD",
    //   "java.rel.annotation.params": "name=\"user_id\",nullable=false",
    //   "java.rel.raw_text": "@Column(name = \"user_id\", nullable = false)"
    // }
    @Column(name = "user_id", nullable = false)
    private Long id;

    // 3. 方法注解
    // Source: Method(save), Target: Class(Transactional)
    // Mores: {
    //   "java.rel.annotation.target": "METHOD",
    //   "java.rel.annotation.params": "timeout=100",
    // }
    @Transactional(timeout = 100)
    // 3.1 参数注解
    // Source: Parameter(data), Target: Class(NotNull)
    // Mores: {
    //   "java.rel.annotation.target": "PARAMETER",
    //   "java.rel.parameter.name": "data",
    // }
    public void save(@NotNull String data) {

        // 4. 局部变量注解
        // Source: Variable(local), Target: Class(NonEmpty)
        // Mores: {
        //   "java.rel.annotation.target": "LOCAL_VARIABLE",
        // }
        @NonEmpty String local = data;
    }
}