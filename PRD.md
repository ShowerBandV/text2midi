# text2midi 项目需求文档 (PRD)

## 项目定位

**AI 驱动的 MIDI 作曲引擎**，不是"生成 MIDI 的工具"，而是"理解音乐、按风格作曲的系统"。

与纯音频 AI（Suno）的根本区别：生成的 MIDI 可编辑、可局部修改、可保留和弦换旋律。

---

## 核心需求

### 1. 文本 → MIDI 生成

- 用户输入自然语言描述（风格/情绪/乐器），输出完整 MIDI 文件
- 支持参数：`--prompt`, `--style`, `--bpm`, `--key`, `--bars`, `--seed`
- 输出格式：SMF Type 1，可直接拖入 FL Studio / Ableton / Logic

### 2. MusicDNA 系统

- 从真实 MIDI 文件中提取结构化信息（和弦进行、动机模式、段落结构）
- 按风格分类存储为 JSON 模板（`templates/jaychou/` 含 45 首）
- Style Profile：从同风格模板中提取统计特征，偏置生成器的随机选择
- 支持 `go run ./scripts/import_dna/` 导入自定义 MIDI

### 3. 4 生成器（Midra 风格 Go 端口）

| 生成器 | 方式 | 文件 |
|--------|------|------|
| `GenerateLeadMidra` | scale-degree 随机动机 + 级进偏置 + 随机力度/时值 | `composer/motif_engine.go` |
| `GenerateBassMidra` | 和弦根音 + 八度跳 + 随机力度/时值 | `composer/song.go` |
| `GenerateChordsMidra` | block/arp 模式交替 + 随机力度/时值 | `composer/song.go` |
| `GenerateDrumsMidra` | kick-snare-hihat + 随机变奏 | `composer/song.go` |

### 4. Composition Plan

- 结构化作曲计划（`plan.json`）：调性、BPM、段落、每段 melody_strategy、音域、密度
- 生成流程：LLM Intent → Plan → Template Lookup → 4 生成器 → Self-Check

### 5. 生成后自检

- 5 项验证：音域范围、力度变化、休止符存在、时值多样性、小节数匹配
- 未通过则提示重生成

---

## 已完成功能清单

- [x] 文本输入 → MIDI 输出（`cmd/generate`）
- [x] LLM Agent：ParseIntent + PlanSong（2 次调用）
- [x] 4 个 Midra 风格生成器（0 次 LLM）
- [x] MusicDNA 提取器（和弦/动机/结构分析）
- [x] 模板库：存储/查询/按风格检索（45 首周杰伦）
- [x] Style Profile：统计特征提取
- [x] Composition Plan（plan.json 每段独立策略）
- [x] 生成后自检（5 项验证）
- [x] MIDI Reader（SMF Type 0/1 解析）
- [x] Stem 导出（分轨 MIDI）
- [x] `--seed` 重现
- [x] ROADMAP / TODO / HARDCODED 文档管理

## 未完成 / 待实现

- [ ] Plan 参数接入生成器（stepProb/velMin/densityF 尚未完全影响输出）
- [ ] 批量模板导入工具化（`import_dna` 目前是脚本，需要 CLI 集成）
- [ ] Web UI / API 服务
- [ ] 更多风格模板（目前只有 jaychou）
- [ ] Soft velocity/timing drift 参数暴露（humanizer 有代码但未接入）

## 不做的

- 不再添加后处理模块（Arranger/Critic/Transition/Dynamic Layering 已证明降低质量）
- 不追求"LLM 生成一切"（Midra 证明规则引擎 + 统计偏置 > 纯 LLM）
- 不做 Composer Personality / DNA 人格系统

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API（默认 DeepSeek）
- **MIDI**: 原生 SMF Type 1 写入/读取器，零外部依赖

## 核心原则

1. 简单 > 复杂（4 个生成器 < 150 行/文件）
2. LLM 只做高层决策（风格/和弦/结构），不做音符生成
3. 统计偏置 > 规则堆砌
4. 生成后验证 > 生成中约束
5. 可编辑 MIDI > 音频输出

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 入口
│   ├── internal/
│   │   ├── agent/                LLM Agent
│   │   ├── composer/             4 个 Midra 生成器 + 动机引擎
│   │   ├── plan/                 Plan 定义 + 生成 + 验证
│   │   ├── musicdna/             MusicDNA 提取 + 模板 + Style Profile
│   │   ├── midi/                 SMF 写入/读取
│   │   ├── llm/                  Prompt 模板
│   │   ├── schema/               数据类型
│   │   └── style/                风格数据库
│   ├── scripts/import_dna/       MIDI → 模板导入
│   ├── templates/jaychou/        45 首 DNA 模板
│   └── midi_output/              已生成 MIDI + plan.json
├── README.md
├── ROADMAP.md
├── HARDCODED.md
└── TODO.md
```

---

## 版本记录

- **V1** (2025-03) — 15 阶段 LLM pipeline，复杂的后处理链
- **V2** (2025-05) — 精简为 4 个生成器，砍掉 Arranger/Critic/Transition
- **V2.1** (2025-06) — MusicDNA + Style Profile + Plan 系统
- **当前** — DNA 驱动的统计风格引擎，4 个轻量生成器 + 自检闭环
