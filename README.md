# text2midi

AI 驱动的 MIDI 作曲引擎。用自然语言描述音乐风格，自动生成完整的可编辑 MIDI 编曲。

纯 Go 实现，零外部依赖。

---

## 快速开始

```bash
export OPENAI_API_KEY=sk-xxxxx

cd backend
go build ./cmd/generate/
go run ./cmd/generate/ \
  --prompt "周杰伦 彩虹 风格, 温暖钢琴叙事曲" \
  --style jaychou \
  --bpm 72 \
  --key "C major" \
  --bars 8 \
  --out ./midi_output
```

参数：`--prompt`, `--style`, `--bpm`, `--key`, `--bars`, `--seed`, `--out`

---

## 架构

```
用户文本 (Prompt)
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent       → 风格/情绪/特征向量
   └── PlanSong          → 和弦/调性/BPM
   ↓
Plan Engine (Go, 0 LLM)
   ├── Template Lookup   → 查询同风格 DNA 模板 → 和弦进行
   ├── Style Profile     → 统计特征 (stepProb/velMin/velMax/densityF/blockRatio)
   └── Plan Builder      → plan.json (每段 melody_strategy / register / density)
   ↓
4 × 生成器 (Go, 0 LLM, 全部参数来自 Style Profile)
   ├── GenerateLeadMidra   → scale-degree motif + 级进偏置 + section density
   ├── GenerateBassMidra   → 和弦根音 + 八度跳 + 随机力度/时值
   ├── GenerateChordsMidra → block/arp 模式交替 + blockRatio 控制
   └── GenerateDrumsMidra  → kick-snare-hihat + densityFactor 控制
   ↓
Validation + Regeneration Loop
   ├── 5 项自检: 音域/力度变化/休止符/时值多样性/小节匹配
   └── 未通过 → 重生成 (最多 3 轮)
   ↓
MIDI Render + MusicDNA 分析
```

**核心原则**: LLM 只做高层决策（风格/和弦/结构），不做音符生成。4 个生成器全部参数化，0 硬编码。

---

## 核心系统

### Style Profile（风格统计引擎）

从同风格 DNA 模板中提取统计特征，直接控制 4 个生成器的所有参数：

| 参数 | 控制 | 来源 |
|------|------|------|
| `stepProb` | Lead 的级进概率 | `profile.IntervalBias` |
| `velMin/velMax` | Lead/Bass 的力度范围 | `profile.VelocityRange` |
| `densityFactor` | Drums 的 hi-hat 密度 | `profile.DensityRange[1]` |
| `blockRatio` | Chords 块式 vs 分解比例 | `profile.BlockVsArpRatio` |

### Plan（作曲计划）

每首曲子先生成 `plan.json`，每段独立参数：

```json
{
  "sections": [
    {"name":"intro",  "bars":2, "energy":0.2, "density":0.3, "register":"low",  "lead_strategy":"new"},
    {"name":"verse",  "bars":2, "energy":0.4, "density":0.5, "register":"low",  "lead_strategy":"variation"},
    {"name":"chorus", "bars":2, "energy":0.8, "density":0.7, "register":"high", "lead_strategy":"development"},
    {"name":"outro",  "bars":2, "energy":0.15,"density":0.2, "register":"low",  "lead_strategy":"recap"}
  ]
}
```

`density` 控制每小节音符数；`register` 控制音区；`lead_strategy` 控制旋律展开方式。

### MusicDNA 模板库

45 首周杰伦 MIDI 被提取为结构化 DNA 模板（`templates/jaychou/`）。每个模板包含和弦进行、动机模式、段落结构。

导入自定义 MIDI：
```bash
cd backend
go run ./scripts/import_dna/ /path/to/midi/files ./templates
```

### 生成后自检

5 项验证（音域/力度变化/休止符/时值多样性/小节匹配），未通过自动重生成，最多 3 轮。

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 入口 + 管道编排
│   ├── internal/
│   │   ├── agent/                LLM Agent (ParseIntent, PlanSong)
│   │   ├── plan/
│   │   │   ├── plan.go           Plan 数据结构 + Build()
│   │   │   └── validate.go       生成后自检 (5 项, max 3 rounds)
│   │   ├── composer/
│   │   │   ├── song.go           Bass/Chords/Drums 生成器
│   │   │   └── motif_engine.go   Lead 生成器 + scale 工具
│   │   ├── musicdna/
│   │   │   ├── types.go          MusicDNA 数据结构
│   │   │   ├── extractor.go      MIDI → DNA 提取器
│   │   │   ├── template.go       模板存储/查询 (45 首 jaychou)
│   │   │   └── profile.go        统计 Style Profile
│   │   ├── midi/
│   │   │   ├── render.go         SMF Type 1 写入
│   │   │   └── reader.go         SMF Type 0/1 读取
│   │   ├── llm/                  Prompt 模板
│   │   ├── schema/               通用数据类型
│   │   └── style/                风格数据库
│   ├── scripts/import_dna/       MIDI → 模板导入工具
│   ├── templates/jaychou/        45 首周杰伦 DNA 模板
│   └── midi_output/              输出 + .clef-work/plan.json
├── frontend/index.html           Web 界面
├── README.md                     本文件
├── PRD.md                        项目需求文档
├── DESIGN.md                     技术方案文档
├── ROADMAP.md                    路线图
├── HARDCODED.md                  硬编码审计
└── TODO.md                       当前待办
```

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生 SMF Type 1 读写器, 零外部依赖

---

## License

MIT
