# text2midi 技术方案文档

## 1. 架构总览

```
用户文本 (Prompt)
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent       → 风格/情绪/特征向量
   └── PlanSong          → 和弦/调性/BPM/结构
   ↓
Plan Engine (Go 规则, 0 LLM)
   ├── Template Lookup   → 查询同风格 DNA 模板 → 和弦进行
   ├── Style Profile     → 统计特征 (和弦偏好/音程偏置/力度范围)
   └── Plan Builder      → plan.json (每段 melody_strategy / register / density)
   ↓
Generation Engine (4 个 Go 规则生成器, 0 LLM)
   ├── Lead Generator    → scale-degree 随机动机 + 级进偏置 + Style Profile 参数
   ├── Bass Generator    → 和弦根音 + 八度跳 + style-aware 音域
   ├── Chords Generator  → block/arp 模式交替 + block_vs_arp ratio from profile
   └── Drums Generator   → kick-snare-hihat + density factor from plan
   ↓
Validation Layer (Go 规则)
   ├── 5 项自检: 音域/力度变化/休止符/时值多样性/小节匹配
   └── 未通过 → 重生成 (最多 3 轮)
   ↓
MIDI Render + MusicDNA 分析
```

### 设计原则

1. **LLM 只做高层决策** — 风格、和弦、结构。不做音符生成
2. **统计偏置 > 规则堆砌** — Style Profile 的统计数据驱动随机选择，不是硬编码阈值
3. **生成后验证 > 生成中约束** — 约束太多会洗掉音乐性
4. **简单 > 复杂** — 每个生成器 < 200 行，职责单一

---

## 2. Plan 引擎

### 2.1 plan.json 数据结构

```json
{
  "format_version": "1.0",
  "tempo": 72,
  "key": {"root": "C", "mode": "major", "scale": "major"},
  "total_bars": 8,
  "sections": [
    {
      "name": "intro",
      "bars": 2,
      "energy": 0.2,
      "density": 0.3,
      "register": "low",
      "lead_strategy": "new"
    },
    {
      "name": "verse",
      "bars": 2,
      "energy": 0.4,
      "density": 0.5,
      "register": "low",
      "lead_strategy": "variation"
    },
    {
      "name": "chorus",
      "bars": 2,
      "energy": 0.8,
      "density": 0.7,
      "register": "high",
      "lead_strategy": "development"
    },
    {
      "name": "outro",
      "bars": 2,
      "energy": 0.15,
      "density": 0.2,
      "register": "low",
      "lead_strategy": "recap"
    }
  ]
}
```

### 2.2 lead_strategy 定义

| 策略 | 含义 | 行为 |
|------|------|------|
| `new` | 全新动机 | 从 scale 随机生成新 motif，锚定首尾 |
| `variation` | 变奏 | 用前一段的 motif，改变音区或节奏 |
| `development` | 展开 | 拆解动机为碎片，逐步加长 |
| `recap` | 再现 | 原样或轻微加花再现第一段动机 |
| `climax` | 高潮 | 组合之前动机碎片，最高音 + 最强力度 |

### 2.3 段落布局（按 total_bars 自适应）

| total_bars | 段落划分 |
|-----------|---------|
| 4 | intro(1) → chorus(2) → outro(1) |
| 8 | intro(2) → verse(2) → chorus(2) → outro(2) |
| 12 | intro(2) → verse(4) → chorus(4) → outro(2) |
| 16+ | intro(2) → verse(4) → pre(2) → chorus(4) → bridge(2) → outro(2) |

---

## 3. Template Lookup + Style Profile

### 3.1 数据流

```
TemplateDB.FindByStyle("jaychou")
   ↓
找到 45 个模板 → 取第一个有和弦的模板
   ↓
chordStrs = ["Em7", "Em7", "Cmaj7", "Csus4", ...]
   ↓
BuildStyleProfile(templates) → 统计特征
   ↓
ChordPreference: [Em7(高), Cmaj7, G, Am, ...]
IntervalBias: [-2, -1, 1, 2, 3, 5]
BlockVsArpRatio: 0.4
VelocityRange: [54, 98]
StepProb: 0.65 + len(IntervalBias)/20
```

### 3.2 Style Profile 参数使用

| 参数 | 使用者 | 作用 |
|------|--------|------|
| `ChordPreference` | Plan Builder | 优先使用高频和弦 |
| `IntervalBias` | Lead Generator | 动机的音程选择范围 |
| `BlockVsArpRatio` | Chords Generator | 块式 vs 分解的概率 |
| `VelocityRange` | Lead/Bass Generator | 力度范围的偏置 |
| `StepProb` | Lead Generator | 级进概率 (替代硬编码 0.65) |

---

## 4. 4 个生成器规范

### 4.1 Lead Generator (`GenerateLeadMidra`)

```
输入: keyRoot, keyMode, totalBars, stepProb, velMin, velMax
输出: []NoteEvent

算法:
1. 从 scale 中随机选 8 个 scale-degree (0-6) → motif
2. 锚定: motif[0]=0, motif[-1]=0/2/4
3. stepProb 概率级进 (±1 or ±2), 否则保留随机值
4. 每 bar 重复 motif, 每音 0.5 拍间隔
5. 力度: velMin + rand(velMax-velMin)
6. 时值: [0.25, 0.4, 0.5, 0.75] 随机 4 选 1
```

**文件**: `internal/composer/motif_engine.go`

### 4.2 Bass Generator (`GenerateBassMidra`)

```
输入: chords[], totalBars
输出: []NoteEvent

算法:
1. 随机 8 个 scale-degree → motif
2. 每 bar 按 chord root 定位
3. 在 beat 0,1,2,3 放置音符
4. scale-degree ≥3 → +12 八度, ≥5 → +24 八度
5. 力度: 82 + rand(23)
6. 时值: [0.5, 0.75, 1.0] 随机 3 选 1
```

**文件**: `internal/composer/song.go`

### 4.3 Chords Generator (`GenerateChordsMidra`)

```
输入: chords[], totalBars
输出: []NoteEvent

算法:
1. 随机 8 个 scale-degree → motif
2. 每 bar 获取当前和弦的 MIDI notes
3. motif[bar%8] in (0,3,6) → block 模式 (长音 2-4 beat)
4. 否则 → arp 模式 (每 1.0 beat 触发)
5. 力度: block 54-78, arp 52-74
6. block_vs_arp ratio 从 Style Profile 读取
```

**文件**: `internal/composer/song.go`

### 4.4 Drums Generator (`GenerateDrumsMidra`)

```
输入: totalBars, densityFactor
输出: []NoteEvent

算法:
1. 随机 hi-hat skip pattern (8 个 0/1)
2. 随机 kick positions from [0.0, 0.75, 1.5, 2.0, 2.75, 3.5]
3. Snare on beat 1.0 and 3.0
4. Hi-hat on 8th notes (skip where pattern=0)
5. 力度: kick 98-116, snare 94-110, hihat 70-96
6. densityFactor 控制 hi-hat skip 密度
```

**文件**: `internal/composer/song.go`

---

## 5. 验证层

### 5.1 自检项

| # | 检查项 | 阈值 | 权重 |
|---|--------|------|------|
| 1 | 小节数匹配 | plan.TotalBars == actual | -0.20 |
| 2 | 音域范围 | 5 < range < 36 半音 | -0.15 |
| 3 | 力度变化 | distinct velocities ≥ 3 | -0.10 |
| 4 | 休止符存在 | 至少 1 bar 音符 < 4 | -0.10 |
| 5 | 时值多样性 | distinct durations ≥ 2 | -0.10 |

### 5.2 重生成策略

```
if score < 0.5:
    重新调用 4 个生成器（使用新随机种子）
    最多 3 轮
    3 轮后仍不通过 → 输出当前结果 + 警告
```

---

## 6. MusicDNA 系统

### 6.1 MIDI → DNA 提取

```
ReadMIDIFile(path)
   ↓
Extract → StructureDNA (段落分界)
       → HarmonyDNA (每 bar 和弦)
       → MotifDNA (interval pattern + frequency scoring)
   ↓
TemplateDB.Save(name, style, DNA) → JSON 文件
```

**文件**: `internal/musicdna/extractor.go`, `internal/musicdna/template.go`

### 6.2 模板库

- 目录结构: `templates/{style}/{name}.json`
- 每个模板包含完整 MusicDNA (motif/harmony/structure/rhythm/emotion)
- `FindByStyle(keyword)` 支持模糊匹配

### 6.3 模板导入

```bash
go run ./scripts/import_dna/ /path/to/midi ./templates
```

自动检测主旋律音轨（avg pitch 60-84），自动分组和弦音符。

---

## 7. 文件组织

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go       CLI 入口 + 管道编排
│   ├── internal/
│   │   ├── agent/                 LLM Agent (ParseIntent, PlanSong)
│   │   ├── plan/
│   │   │   ├── plan.go            Plan 数据结构 + Build()
│   │   │   └── validate.go        生成后自检 (5 项)
│   │   ├── composer/
│   │   │   ├── song.go            4 个 Midra 生成器 (bass/chords/drums)
│   │   │   └── motif_engine.go   Lead 生成器 + scale 工具
│   │   ├── musicdna/
│   │   │   ├── types.go           MusicDNA 数据结构
│   │   │   ├── extractor.go       MIDI → DNA 提取器
│   │   │   ├── template.go        模板存储/查询
│   │   │   └── profile.go         Style Profile 统计分析
│   │   ├── midi/
│   │   │   ├── render.go          SMF Type 1 写入
│   │   │   └── reader.go          SMF Type 0/1 读取
│   │   ├── llm/                   Prompt 模板
│   │   ├── schema/                通用数据类型
│   │   └── style/                 风格数据库
│   ├── scripts/import_dna/        MIDI → 模板导入工具
│   ├── templates/jaychou/         45 首周杰伦 DNA 模板
│   └── midi_output/               输出 + .clef-work/plan.json
├── frontend/index.html            Web 界面
├── PRD.md                         项目需求文档
├── ROADMAP.md                     路线图
├── HARDCODED.md                   硬编码审计
└── TODO.md                        当前待办
```

---

## 8. 与 Clef 的对比

| 维度 | Clef | text2midi |
|------|------|-----------|
| **中间格式** | ABC 记谱法 | MusicDNA (JSON) |
| **生成方式** | 多 Agent LLM (7 个) | 4 个 Go 规则生成器 |
| **知识库** | theory-* skills (6 个) | Style Profile 统计 |
| **模板库** | 无 | 45 首周杰伦 DNA |
| **自检** | music21 + Reviewer Agent | Go 规则 5 项 |
| **迭代** | Leader + Reviewer, ≤ 3 轮 | 自检触发重生成, ≤ 3 轮 |
| **表现力** | Orchestrator Agent → CC/弯音 | 无 (待实现) |
| **技术栈** | Python + Godot | 纯 Go |

**我们独有的**: DNA 模板库 + Style Profile 统计 + 纯 Go 无外部依赖

**Clef 值得借鉴但暂不实现**: ABC 中间格式、多 Agent 迭代、Orchestrator 表现力层

---

## 9. 技术决策记录

| 决策 | 原因 |
|------|------|
| **砍掉 Arranger/Critic/Transition** | 这些模块给 LLM 生成的旋律做了太多修正，实际上在洗掉音乐性 |
| **砍掉 Composer Personality** | 人格系统在规则引擎下有意义，但 Midra 风格生成器不依赖它 |
| **保 DNA 提取器** | 这是我们的核心差异化 — 从真实 MIDI 学习 |
| **保 Style Profile** | 统计偏置 > 规则堆砌，这是做出风格差异的关键 |
| **不引入 ABC** | MusicDNA JSON 已经提供了结构化表示，ABC 增加了不必要的复杂性 |
