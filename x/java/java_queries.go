package java

// JavaActionQuery 定义了核心动作的 Tree-sitter 查询语句。
// 约定：每个模式的第一个捕获位（@xxx_stmt）用于确定关系的 Source 和 Context。
const JavaActionQuery = `
[
  ; 1. 方法调用 (Call)
  (method_invocation name: (identifier) @call_target) @call_stmt
  
  ; 2. 方法引用 (Functional Call)
  (method_reference (identifier) @ref_target) @ref_stmt
  
  ; 3. 对象与数组创建 (Create)
  (object_creation_expression
    type: [
        (type_identifier) @create_target 
        (generic_type (type_identifier) @create_target)
    ]) @create_stmt

  (array_creation_expression
    type: (type_identifier) @create_target) @create_stmt

  ; 4. 显式构造函数调用 (super/this)
  (explicit_constructor_invocation) @explicit_constructor_stmt

  ; 5. 字段访问 (Use)
  ; --普通标识符读取, 后续过滤
  (identifier) @id_atom

  ; 6. 赋值动作 (Assign)
  (assignment_expression 
    left: [
        (identifier) @assign_target
        (field_access field: (identifier) @assign_target)
        (array_access array: (identifier) @assign_target)
    ]) @assign_stmt

  ; --自增/自减 (Assign)
  (update_expression 
    [
        (identifier) @assign_target
        (field_access field: (identifier) @assign_target)
    ]) @update_stmt

  ; 7. 变量声明中的初始化赋值 (Assign/Create)
  (variable_declarator 
    name: (identifier) @assign_target
    value: (_) @assign_value) @variable_stmt

  ; 8. 抛出异常 (Throw)
  (throw_statement
    [
      (object_creation_expression 
        type: [
          (type_identifier) @throw_target 
          (generic_type (type_identifier) @throw_target)
        ])
      (identifier) @throw_target
    ]
  ) @throw_stmt
]
`
