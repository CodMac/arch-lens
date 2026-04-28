
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
# 克隆仓库
git clone https://github.com/CodMac/arch-lens.git

# 创建技能存放目录
mkdir -p ~/.claude/skills/archLens-sentinel/references

# 拷贝缺陷协议
cp -R arch-lens/doc/    ~/.claude/skills/archLens-sentinel/references/

# 拷贝技能定义文件
cp arch-lens/skills/archLens-sentinel/SKILL.md    ~/.claude/skills/archLens-sentinel/SKILL.md
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

本技能采用 **点名触发机制**，所有核心指令需以 `/archLens-sentinel` 开头。Agent 将严格执行协议同步与分治扫描逻辑。

| 触发场景 | 指令示例 | 指令逻辑说明 |
| :--- | :--- | :--- |
| **主动扫描模式** | `/archLens-sentinel @待分析源码清单 [./src] [上帝类]` | 扫描指定路径。若路径过载，将自动触发**热点探测**。 |
| **全量健康审计** | `/archLens-sentinel @待分析源码清单 [./src] [ALL]` | 逐项执行 19 项协议扫描，产生的中间件 JSON 将落盘至 `~/.archlens/`。 |
| **基于结果分析** | `/archLens-sentinel @分析结果 [~/.archlens/task_xxx/tmp_smell_07.json]` | **强制读取**本地落盘数据，进行量化指标对撞诊断，严禁凭空生成。 |
| **第三方复核** | `/archLens-sentinel @三方结果 [sonar_report.json]` | 导入第三方分析报告，利用内置 19 项协议进行二次校准评估。 |
| **定向修复设计** | `/archLens-sentinel @设计修复方案 [#07/UserService.java]` | 参考本地协议快照中的治理建议，出具带有指标预期贡献的“重构蓝图”。 |
| **原子化提交** | `/archLens-sentinel @执行重构` | 在确认方案无误后，授权 Agent 执行物理文件写入。 |

---

## 📚 支撑协议库：19 个架构缺陷类型
Agent 内部逻辑严格对应以下 19 项协议，分析时将输出对应的协议编号、判定指标（如 $WMC$, $LCOM4$）及大白话说明：

| 编号 | 分类 | 核心缺陷项 | 判定核心指标 |
| :--- | :--- | :--- | :--- |
| **#01-#03** | **依赖风险** | [循环依赖](https://github.com/CodMac/arch-lens/blob/main/doc/%E5%BE%AA%E7%8E%AF%E4%BE%9D%E8%B5%96%20(Circular%20Dependency).md) / [不稳定依赖](https://github.com/CodMac/arch-lens/blob/main/doc/%E4%B8%8D%E7%A8%B3%E5%AE%9A%E4%BE%9D%E8%B5%96(Stable%20Dependencies%20Principle%2C%20SDP).md) / [破坏稳定抽象](https://github.com/CodMac/arch-lens/blob/main/doc/%E7%A0%B4%E5%9D%8F%E7%A8%B3%E5%AE%9A%E6%8A%BD%E8%B1%A1%E5%8E%9F%E5%88%99%20(SAP).md) | $CCD$ / $I_{dep}$ / $D$ 值 |
| **#04-#06** | **继承风险** | [拒绝父类馈赠](https://github.com/CodMac/arch-lens/blob/main/doc/%E6%8B%92%E7%BB%9D%E7%88%B6%E7%B1%BB%E9%A6%88%E8%B5%A0%20(Refused%20Bequest).md) / [传统破坏者](https://github.com/CodMac/arch-lens/blob/main/doc/%E4%BC%A0%E7%BB%9F%E7%A0%B4%E5%9D%8F%E8%80%85%20(Tradition%20Breaker).md) / [继承层次混乱](https://github.com/CodMac/arch-lens/blob/main/doc/%E7%BB%A7%E6%89%BF%E5%B1%82%E6%AC%A1%E6%B7%B7%E4%B9%B1%20(Hierarchy%20Complexity).md) | $BUR$ / $NAS$ / $DIT$ |
| **#07-#12** | **职责内聚** | [上帝类](https://github.com/CodMac/arch-lens/blob/main/doc/%E4%B8%8A%E5%B8%9D%E7%B1%BB%20(God%20Class).md) / [上帝文件](https://github.com/CodMac/arch-lens/blob/main/doc/%E4%B8%8A%E5%B8%9D%E6%96%87%E4%BB%B6%20(God%20File).md) / [复杂类](https://github.com/CodMac/arch-lens/blob/main/doc/%E5%A4%8D%E6%9D%82%E7%B1%BB%20(Complex%20Class).md) / [精神分裂类](https://github.com/CodMac/arch-lens/blob/main/doc/%E7%B2%BE%E7%A5%9E%E5%88%86%E8%A3%82%E7%B1%BB%20(Schizophrenic%20Class).md) | $WMC$ / $LCOM4$ / $AMCC$ |
| **#13-#17** | **耦合/解耦** | [违反迪米特法则](https://github.com/CodMac/arch-lens/blob/main/doc/%E8%BF%9D%E5%8F%8D%E8%BF%AA%E7%B1%B3%E7%89%B9%E6%B3%95%E5%88%99%20(Violation%20of%20Law%20of%20Demeter%2C%20LoD).md) / [消息链](https://github.com/CodMac/arch-lens/blob/main/doc/%E6%B6%88%E6%81%AF%E9%93%BE%20(Message%20Chain).md) / [依恋情节](https://github.com/CodMac/arch-lens/blob/main/doc/%E5%8A%9F%E8%83%BD%E4%BE%9D%E6%81%8B%20(Feature%20Envy).md) | $MCL$ / $ATFD$ / $CINT$ |
| **#18-#19** | **建模缺陷** | [数据类](https://github.com/CodMac/arch-lens/blob/main/doc/%E6%95%B0%E6%8D%AE%E7%B1%BB%20(Data%20Class).md) / [数据泥团](https://github.com/CodMac/arch-lens/blob/main/doc/%E6%95%B0%E6%8D%AE%E6%B3%A5%E5%9B%A2%20(Data%20Clumps).md) | $WMC$ / $PGF$ |

---

### 🔄 约束性工作流 (Agent Workflow)
Agent 将自动执行以下四个阶段，用户可通过观察 **[工序状态]** 跟踪进度：
1. **识别 (Recognition)**：同步协议快照 $\to$ 分治原子扫描 $\to$ 结果落盘。
2. **诊断 (Diagnosis)**：指标对撞 $\to$ 架构健康报告汇总。
3. **蓝图 (Blueprinting)**：重构方案设计 $\to$ 可视化 Diff 预览。
4. **执行 (Execution)**：原子化代码提交 $\to$ 自动化指标回检。

---

**ArchLens-Sentinel: 赋予命令行工具真正的架构审查灵魂。**
