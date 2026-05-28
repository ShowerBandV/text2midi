# text2midi

AI 驱动的 MIDI 作曲引擎。用自然语言描述想要的音乐风格，自动生成完整的可编辑 MIDI 编曲。

纯 Go 实现，零外部依赖，编译为静态二进制文件。

---

## 快速开始

```bash
# 前置条件：设置 API Key
export OPENAI_API_KEY=sk-xxxxx

# 构建
cd backend
go build ./cmd/generate/

# 生成一首 MIDI
go run ./cmd/generate/ \
  --prompt "周杰伦 晴天 风格, 流行钢琴" \
  --style lofi \
  --bpm 85 \
  --key "C major" \
  --bars 8 \
  --out ./midi_output

# 运行全部测试
go test ./...
```

---

## 核心架构

```
用户文本 (Prompt)
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent → 风格/情绪/特征向量
   └── PlanSong    → 和弦/调性/BPM
   ↓
V2 作曲引擎 (Go 规则引擎, 0 LLM 调用)
   ├── Planner     → 结构化 SongPlan（段落/能量/配器）
   ├── Phrase      → 4-bar 乐句展开（风格感知）
   ├── Composer    → 多轨编曲（bass/drums/lead/pad/fx）
   ├── Arranger    → 编曲协调（音区冲突/密度控制）
   ├── Critic      → 质量评估（4 维评分 + 重生成判断）
   ├── Humanizer   → 人性化（velocity/timing drift）
   └── MusicDNA    → 结构化分析报告
   ↓
Post-processing
   ├── MelodyGrammar   → 调性/音程约束
   ├── Transition Engine → 段落过渡
   ├── Dynamic Layering  → 乐器数量控制
   ├── Texture Layer     → 氛围铺底
   └── Creative Chaos    → 故意犯错
   ↓
MIDI → .mid 文件 + MusicDNA 分析
```

**核心原则**：
- LLM 只做高层决策（风格、情绪、和弦），所有编曲由 Go 规则引擎确定性执行
- 不是"生成 MIDI 再修修补补"，而是"先规划结构再生长出完整作品"
- 每次生成附带 MusicDNA 分析报告（段落/和弦/动机/节奏/织体/情绪）

---

## V2 架构详解

### 1. Planner（歌曲规划器）

决定歌曲的「骨架」—— 输入特征向量，输出结构化 SongPlan。

```
输入:  feature_vector (darkness/energy/rhythmic/tension)
输出:  SongPlan { BPM, Key, Sections[SectionPlan] }

每个 SectionPlan:
  Name:     "intro" / "verse" / "chorus" / "bridge" / "outro"
  Bars:     2 / 4 / 8
  Energy:   0.2 / 0.5 / 0.9
  Density:  0.2 / 0.5 / 0.8
  Instruments: ["piano"] / ["drums", "bass", "lead"] / ["all"]
  MotifMode: "sparse" / "partial" / "full" / "invert"
```

风格感知的段落布局：

| 风格 | 触发条件 | 段落结构 |
|------|---------|---------|
| **Metal** | dark>0.7, energy>0.7 | intro(1) → verse(2) → chorus(4) → bridge(1) |
| **Pop** | energy>0.4, rhythmic<0.5 | intro(2) → verse(4) → pre(2) → chorus(4) → bridge(2) → outro(2) |
| **Hip-hop** | rhythmic>0.5, energy>0.3 | intro(1) → loop_a(3) → loop_b(3) → outro(1) |
| **Ambient** | default | intro(2) → verse(4) → chorus(4) → outro(2) |

### 2. Phrase（乐句系统）

音乐不是按 bar 生成的，是按 **phrase**（4 小节乐句）。

```
每个 phrase = 4 bars = question (bars 1-2) + answer (bars 3-4)
```

风格感知的句式：

| 风格 | Bar 0 | Bar 1 | Bar 2 | Bar 3 |
|------|-------|-------|-------|-------|
| **Metal** | Riff | 八度下移 | Riff 重复 | 碎片+变奏 |
| **Pop** | A (motif) | A' (+3 移调) | B (对比) | A (回归) |
| **Hip-hop** | Loop | Loop | Loop | Loop (+3 变奏) |
| **Ambient** | 2 音 | +2 移调 | +4 移调 | +6 移调 |

### 3. Composer（作曲系统）

从 Planner 的 SongPlan + Phrase 的乐句展开完整多轨编曲：

```
Timeline Planner → 风格感知的段落结构
Motif Allocation → 每段用哪种动机变体
Drums Generator  → 风格感知的鼓模式 (4 种)
Bass Generator   → 风格感知的贝斯线 (metal/pop/hiphop/ambient)
Pad Generator    → 风格感知的和弦铺底
Humanizer        → velocity drift / timing drift / ghost note
```

### 4. Arranger（编曲协调器）

确保各轨道不打架。

检测以下冲突并自动修复：

| 冲突类型 | 检测条件 | 修复方法 |
|---------|---------|---------|
| **音区冲突** | bass 和 lead 在同一中频区 | 贝斯降八度 |
| **密度过载** | 单小节总音符 > 32 | 降低部分轨道力度 |
| **空小节** | 没有任何音符 | 自动填充 |
| **节奏冲突** | 鼓密集 + 旋律也密集 | 旋律简化 |

### 5. Critic（质量评估器）

生成后对音乐打分，低分触发重生成。

```
Score = repetition * 0.25 + tension * 0.2 + groove * 0.2 + climax * 0.2 + density * 0.15
```

| 维度 | 检测方法 | 意义 |
|------|---------|------|
| **Repetition** | 旋律中 3-note 模式重复次数 | 够不够"记得住" |
| **Tension** | 各小节能量方差 | 有无起伏 |
| **Groove** | 鼓 velocity 标准差 | 节奏有没有"味道" |
| **Climax** | 后 1/3 与前 1/3 能量比 | 副歌够不够"炸" |
| **Density** | 各小节音符数极差 | 段落对比够不够 |

`if score.Total < 0.4 → 触发重生成`

### 6. Humanizer（人性化系统）

让 MIDI 听感更接近真人演奏：

- **Velocity Drift**: 每音符力度 ± 随机偏移
- **Timing Drift**: 每音符起始时间 ± 随机偏移
- **Ghost Notes**: 弱拍随机添加极轻的幽灵音
- **Accent Beat**: 每小节第一拍加重

### 7. MusicDNA（结构化音乐分析）

每首 MIDI 输出完整分析报告：

```
===== MusicDNA =====
--- Structure ---   段落：intro/verse/chorus, 能量/密度
--- Harmony ---     调性 + 每小节和弦
--- Motif ---       动机 interval pattern + 四项评分
--- Rhythm ---      密度/swing/切分
--- Texture ---     每轨音域/角色/音符数
--- Dynamics ---    力度范围/平均值
--- Emotion ---     tension/energy/warmth/stability/brightness
```

### 8. ComposerDNA（作曲家人格）

8 种内置人格，影响动机重复率、混沌程度、声部交叉规则：

| 人格 | MotifObsession | Chaos | 适用场景 |
|------|----------------|-------|---------|
| **Toby Fox (Undertale)** | 0.90 | 0.30 | 游戏音乐 |
| **Nobuo Uematsu (FF)** | 0.70 | 0.10 | JRPG |
| **Hans Zimmer** | 0.85 | 0.20 | 史诗/电影 |
| **Hyperpop Maniac** | 0.40 | 0.90 | 实验 |
| **Classical Purist** | 0.60 | 0.05 | 古典 |

---

## 完整生成流程 (17 阶段)

| 阶段 | 模块 | 说明 |
|------|------|------|
| 1 | ParseIntent | LLM 解析文本 → 风格/情绪/特征向量 |
| 2 | PlanSong | LLM 生成和弦 + 调性 + BPM |
| 3 | **Planner** | 结构化 SongPlan（段落/能量/配器） |
| 4 | **Phrase** | 4-bar 乐句展开（按风格选句式） |
| 5 | Composer | 多轨编曲（bass/drums/lead/pad/fx） |
| 6 | Humanizer | velocity/timing drift + ghost note |
| 7 | **Arranger** | 编曲协调（音区冲突/密度控制） |
| 8 | **Critic** | 质量评估（低分触发重生成） |
| 9 | MusicDNA | 结构化分析报告 |
| 10 | MelodyGrammar | 调性/音程约束 |
| 11 | Transition Engine | 段落过渡 (6 种类型) |
| 12 | Dynamic Layering | 乐器数量按能量控制 |
| 13 | Texture Layer | 氛围铺底 |
| 14 | Creative Chaos | 故意犯错 |
| 15 | MIDI Render | .mid 文件 |

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 生成器
│   ├── internal/
│   │   ├── agent/                LLM Agent (意图/和弦/配器)
│   │   ├── planner/              V2: 歌曲规划器 (SongPlan)
│   │   ├── phrase/               V2: 乐句系统 (4-bar 句式)
│   │   ├── arranger/             V2: 编曲协调器 (冲突检测)
│   │   ├── critic/               V2: 质量评估器 (4 维评分)
│   │   │
│   │   ├── composer/             作曲引擎
│   │   │   ├── song.go           Song Composer
│   │   │   ├── emotion.go        Emotion Engine
│   │   │   ├── motif_engine.go   Motif Engine
│   │   │   ├── humanizer.go      Humanizer
│   │   │   ├── phrase.go         风格感知句式
│   │   │   ├── melody.go         MelodyGrammar
│   │   │   ├── transition.go     Transition Engine
│   │   │   ├── dynamics.go       Dynamic Layering
│   │   │   ├── texture.go        Texture Layer
│   │   │   ├── groove.go         Swing 量化
│   │   │   ├── energy.go         能量曲线
│   │   │   ├── stems.go          Stem 导出
│   │   │   ├── substitution.go   和弦代换
│   │   │   └── dna.go            ComposerDNA + SongMemory
│   │   │
│   │   ├── musicdna/             结构化音乐分析
│   │   │   ├── types.go          MusicDNA 数据模型
│   │   │   └── extractor.go      MIDI→DNA 提取器
│   │   │
│   │   ├── generator/            规则乐器生成器
│   │   ├── llm/                  Prompt 模板
│   │   ├── midi/                 原生 SMF Type 1 写入器
│   │   ├── music/                乐理工具
│   │   ├── mutation/             Creative Chaos
│   │   ├── schema/               数据类型
│   │   └── style/                风格数据库
│   │
│   ├── 编曲方法/                  编曲知识库
│   └── midi_output/              已生成 MIDI
│
├── frontend/index.html           Web 界面
├── ROADMAP.md                    路线图
├── HARDCODED.md                  硬编码审计
└── TODO.md                       当前待办
```

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生写入，零外部依赖
- **前端**: 纯 HTML/JS (无框架)

## License

MIT
