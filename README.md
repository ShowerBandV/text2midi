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

## 架构总览

```
用户文本 (Prompt)
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent → 风格/情绪/特征向量
   └── PlanSong    → 和弦进行/调性/BPM
   ↓
MusicDNA Core
   ├── Emotion Engine    → 情绪→音乐参数映射
   ├── Motif Engine      → 动机展开 (A-A'-B-A)
   ├── Song Composer     → 多轨编曲
   ├── Humanizer         → 人性化 (velocity/timing drift)
   └── MusicDNA Writer   → 结构化分析报告
   ↓
Post-processing
   ├── MelodyGrammar     → 调性/音程约束
   ├── Transition Engine → 段落过渡
   ├── Dynamic Layering  → 乐器数量控制
   ├── Texture Layer     → 氛围铺底
   └── Creative Chaos    → 故意犯错
   ↓
MIDI → .mid 文件 + MusicDNA 分析
```

**核心原则**：
- LLM 只做高层决策（风格、情绪、和弦），所有编曲由 Go 规则引擎确定性执行
- 不是"生成音乐再修修补补"，而是"从 DNA 生长出完整作品"
- 每次生成附带 MusicDNA 结构化分析报告

---

## 核心系统

### 1. Emotion Engine（情绪引擎）

将自然语言情绪映射为 5 维音乐参数：

| 维度 | 范围 | 作用 |
|------|------|------|
| **Tension** | 0-1 | 紧张度 → 和弦复杂度 |
| **Energy** | 0-1 | 能量 → 节奏密度 |
| **Warmth** | 0-1 | 温暖度 → 音色选择 |
| **Stability** | 0-1 | 稳定度 → 动机重复率 |
| **Brightness** | 0-1 | 明亮度 → 音区控制 |

### 2. Motif Engine（动机引擎）

用 3-8 音的短动机，通过风格感知的句式展开全曲旋律：

| 风格 | 句式 | 特征 |
|------|------|------|
| **Metal** | Riff 重复 + 八度下移 | 重复 riff, 最小变奏 |
| **Pop** | A-A'-B-A | 动机→变奏→对比→回归 |
| **Hip-hop** | Loop 循环 + 渐变 | 4 小节循环, 每轮微变 |
| **Ambient** | 稀疏渐进 | 每乐句 +2 半音演进 |

5 种变奏方法：Transpose / Invert / Retrograde / Fragment / Extend。

### 3. Song Composer（作曲系统）

从动机 + 和弦 + 情绪展开完整多轨编曲：

```
Timeline Planner → 风格感知的段落结构
Motif Allocation → 每段用哪种动机变体
Drums Generator → 风格感知的鼓模式 (4 种)
Bass Generator  → 风格感知的贝斯线 (metal/pop/hiphop/ambient)
Pad Generator   → 风格感知的和弦铺底
Humanizer       → velocity drift / timing drift / ghost note
```

### 4. Humanizer（人性化系统）

让 MIDI 听感更接近真人演奏：

- **Velocity Drift**: 每音符力度 ± 随机偏移
- **Timing Drift**: 每音符起始时间 ± 随机偏移
- **Ghost Notes**: 弱拍随机添加极轻的幽灵音
- **Accent Beat**: 每小节第一拍加重

### 5. MusicDNA（结构化音乐分析）

每首 MIDI 输出完整的分析报告：

```
===== MusicDNA =====
--- Structure ---  bars 0-7, energy, density
--- Harmony ---    Key, chord progression per bar
--- Motif ---      Pattern (intervals), Score (repetition/contour/simplicity/rhythm)
--- Rhythm ---     density, swing, syncopation
--- Texture ---    track roles, note counts, avg pitch
--- Dynamics ---   velocity range, avg velocity
--- Emotion ---    tension, energy, warmth, stability, brightness
```

### 6. ComposerDNA（作曲家人格）

8 种内置作曲家人格，影响动机重复率、混沌程度、声部交叉规则：

| 人格 | MotifObsession | Chaos | 适用场景 |
|------|----------------|-------|---------|
| **Toby Fox (Undertale)** | 0.90 | 0.30 | 游戏音乐 |
| **Nobuo Uematsu (FF)** | 0.70 | 0.10 | JRPG |
| **Hans Zimmer** | 0.85 | 0.20 | 史诗/电影 |
| **Hyperpop Maniac** | 0.40 | 0.90 | 实验 |
| **Classical Purist** | 0.60 | 0.05 | 古典 |

### 7. Motif Scoring（旋律质量评估）

四项加权评分，驱动生成决策：

```
Score = repetition * 0.4 + contour * 0.2 + simplicity * 0.2 + rhythm * 0.2
```

高分 → 多重复、少变化；低分 → 强变异、打破模式。

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 生成器
│   ├── internal/
│   │   ├── agent/                LLM Agent (意图/和弦/配器)
│   │   ├── composer/
│   │   │   ├── emotion.go        Emotion Engine (情绪映射)
│   │   │   ├── song.go           Song Composer + Section-aware
│   │   │   ├── phrase.go         风格感知的句式生成
│   │   │   ├── motif_engine.go   Motif Engine (变奏/展开)
│   │   │   ├── humanizer.go      Humanizer (velocity/timing drift)
│   │   │   ├── melody.go         MelodyGrammar (调性约束)
│   │   │   ├── transition.go     Transition Engine (段落过渡)
│   │   │   ├── dynamics.go       Dynamic Layering (乐器数量)
│   │   │   ├── texture.go        Texture Layer (氛围铺底)
│   │   │   ├── groove.go         Swing 量化 + 乐句呼吸
│   │   │   ├── energy.go         能量曲线 + Bass-Kick 对齐
│   │   │   ├── stems.go          Stem 分轨导出
│   │   │   ├── substitution.go   和弦代换
│   │   │   └── dna.go            ComposerDNA + SongMemory
│   │   ├── musicdna/
│   │   │   ├── types.go          MusicDNA 数据模型
│   │   │   └── extractor.go      MIDI→DNA 提取器 (segmenter/chord/motif)
│   │   ├── generator/            规则乐器生成器
│   │   ├── llm/                  Prompt 模板 + LLM 客户端
│   │   ├── midi/                 原生 SMF Type 1 写入器
│   │   ├── music/                乐理工具
│   │   ├── mutation/             Creative Chaos
│   │   ├── schema/               核心数据类型
│   │   └── style/                风格数据库 + 配器模板
│   │
│   ├── 编曲方法/                  编曲知识库 (流行/摇滚/说唱)
│   └── midi_output/              已生成 MIDI 文件
│
├── frontend/index.html           Web 界面
├── ROADMAP.md                    7 阶段路线图
├── HARDCODED.md                  硬编码审计清单
└── TODO.md                       当前待办
```

---

## 生成流程 (17 阶段)

| 阶段 | 模块 | 说明 |
|------|------|------|
| 1 | ParseIntent | LLM 解析文本 → 风格/情绪/特征向量 |
| 2 | PlanSong | LLM 生成和弦 + 调性 + BPM |
| 3 | EmotionEngine | 情绪 → Tension/Energy/Warmth/Stability/Brightness |
| 4 | Timeline Planner | 风格感知的段落结构 |
| 5 | Motif Engine | 动机变奏展开 (按风格选句式) |
| 6 | Bass Generator | 风格感知贝斯线 |
| 7 | Drums Generator | 风格感知鼓模式 |
| 8 | Lead Melody | 动机展开 + 人性化 |
| 9 | Pad Generator | 和弦铺底 |
| 10 | Humanizer | velocity/timing drift + ghost note |
| 11 | Transition Engine | 段落过渡 (6 种类型) |
| 12 | Dynamic Layering | 乐器数量按能量控制 |
| 13 | Texture Layer | 氛围铺底 |
| 14 | MelodyGrammar | 调性/音程几何约束 |
| 15 | Creative Chaos | 故意犯错 |
| 16 | MusicDNA Extractor | 结构化分析 |
| 17 | MIDI Render | .mid 文件 |

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生写入, 零外部依赖
- **前端**: 纯 HTML/JS (无框架)

## License

MIT
