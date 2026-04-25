
---

# ArchLens-Sentinel (架构哨兵) Agent Skill

**ArchLens-Sentinel** 是一款专为 `claudecode` 与 `gemini-cli` 打造的工业级架构治理 Agent 技能。它将 [Arch-Lens 19 项架构协议](https://github.com/CodMac/arch-lens/tree/main/doc) 深度集成到 AI 的逻辑链路中，使其具备自动审计、量化评估和受控重构的能力。

---

## 🚀 核心价值
* **协议驱动**：强制 AI 按照 19 项专业架构分析协议（非模糊感性判断）审视代码。
* **量化诊断**：通过 Tree-sitter 等工具感知 $LCOM$, $WMC$, $CogC$ 等指标。
* **受控重构**：遵循“识别-方案-审核-执行”的严格闭环，确保生产安全。

---

## 🛠️ 安装与启动

本技能采用 **本地 Agent Skill** 模式安装，无需配置复杂的 MCP 服务。

### 1. 下载并安装
将技能协议文件拷贝到 AI 工具的全局技能目录：

```bash
# 创建技能存放目录
mkdir -p ~/.agents/skills/

# 将技能定义文件下载/拷贝至该目录
cp ./skills/archLens-sentinel.md ~/.agents/skills/
```

### 2. 进入源码环境
在终端中进入您需要分析的工程根目录：
```bash
cd /your/project/path
```

### 3. 启动工具
启动支持 Agent Skills 的 AI CLI：
```bash
claude  # 或使用相应的 gemini-cli 封装
```

---

## 📖 使用指令 (Trigger)

启动后，您可以通过以下指令触发特定的架构治理流程：

| 触发场景 | 指令示例 |
| :--- | :--- |
| **主动扫描模式** | `@archLens-sentinel 扫描当前源码，识别缺陷。` |
| **报告对齐模式** | `@archLens-sentinel 解析此报告 [粘贴结果]，进行协议复核。` |
| **定向重构设计** | `针对发现的 #07 上帝类问题，请按照协议设计修复方案。` |
| **受控执行修改** | `方案已确认，请执行重构。` |

---

## 📚 支撑协议库：19 个架构缺陷类型
Agent 内部逻辑严格对应以下 19 项协议，分析时将输出对应的协议编号：

| 编号 | 缺陷名称 | 协议文档链接 (Strict Reference) |
| :--- | :--- | :--- |
| **01-03** | **依赖风险** | [循环依赖](https://github.com/CodMac/arch-lens/blob/main/doc/01.%E5%BE%AA%E7%8E%AF%E4%BE%9D%E8%B5%96.md) / [不稳定依赖](https://github.com/CodMac/arch-lens/blob/main/doc/02.%E4%B8%8D%E7%A8%B3%E5%AE%9A%E4%BE%9D%E8%B5%96.md) / [破坏稳定抽象](https://github.com/CodMac/arch-lens/blob/main/doc/03.%E7%A0%B4%E5%9D%8F%E7%A8%B3%E5%AE%9A%E6%8A%BD%E8%B1%A1%E5%8E%9F%E5%88%99.md) |
| **04-06** | **继承风险** | [拒绝父类馈赠](https://github.com/CodMac/arch-lens/blob/main/doc/04.%E6%8B%92%E7%BB%9D%E7%88%B6%E7%B1%BB%E9%A6%88%E8%B5%A0.md) / [传统破坏者](https://github.com/CodMac/arch-lens/blob/main/doc/05.%E4%BC%A0%E7%BB%9F%E7%A0%B4%E5%9D%8F%E8%80%85.md) / [继承层次混乱](https://github.com/CodMac/arch-lens/blob/main/doc/06.%E7%BB%A7%E6%89%BF%E5%B1%82%E6%AC%A1%E6%B7%B7%E4%B9%B1.md) |
| **07-12** | **职责内聚** | [上帝类](https://github.com/CodMac/arch-lens/blob/main/doc/07.%E4%B8%8A%E5%B8%9D%E7%B1%BB%20(God%20Class).md) / [复杂文件](https://github.com/CodMac/arch-lens/blob/main/doc/10.%E5%A4%8D%E6%9D%82%E6%96%87%E4%BB%B6%20(Blob%20File).md) / [精神分裂类](https://github.com/CodMac/arch-lens/blob/main/doc/11.%E7%B2%BE%E7%A5%9E%E5%88%86%E8%A3%82%E7%B1%BB%20(Schizophrenic%20Class).md) |
| **13-17** | **耦合/解耦** | [违反迪米特法则](https://github.com/CodMac/arch-lens/blob/main/doc/13.%E8%BF%9D%E5%8F%8D%E8%BF%AA%E7%B1%B3%E7%89%B9%E6%B3%95%E5%88%99.md) / [消息链](https://github.com/CodMac/arch-lens/blob/main/doc/14.%E6%B6%88%E6%81%AF%E9%93%BE%20(Message%20Chain).md) / [依恋情节](https://github.com/CodMac/arch-lens/blob/main/doc/16.%E4%BE%9D%E6%81%8B%E6%83%85%E8%8A%82%20(Feature%20Envy).md) |
| **18-19** | **建模缺陷** | [数据类](https://github.com/CodMac/arch-lens/blob/main/doc/18.%E6%95%B0%E6%8D%AE%E7%B1%BB%20(Data%20Class).md) / [数据泥团](https://github.com/CodMac/arch-lens/blob/main/doc/19.%E6%95%B0%E6%8D%AE%E6%B3%A5%E5%9B%A2%20(Data%20Clumps).md) |

---

## 🧩 差异化治理策略 (Differentiation)
Agent 安装后将自动执行环境感知：
* **Go 项目**：自动跳过 Java 风格的继承类检查（#04-#06），强制开启 **#12 精神分裂文件** 审计。
* **Java 项目**：开启全量协议审计，强化对 **#03 稳定抽象原则** 的验证。
* **大型重构**：针对战略级缺陷（如循环依赖），Agent 优先提供“架构拓扑图”调整建议，而非直接改动局部代码。

---

## ⚖️ 安全护栏
1. **语义保全**：严禁修改任何业务算子逻辑。
2. **原子化提交**：确保跨文件重构的完整性。
3. **用户主权**：未获得交互式确认 `@执行重构` 前，AI 不得执行任何文件写入操作。

---

> **ArchLens-Sentinel: 赋予 AI 命令行工具真正的架构审查灵魂。**
