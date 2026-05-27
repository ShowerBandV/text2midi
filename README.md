# text2midi

AI 驱动的 MIDI 音乐生成引擎。用自然语言描述想要的音乐风格，自动生成完整的 MIDI 编曲。

纯 Go 实现，零外部依赖，编译为静态二进制文件。

---

## 快速开始

```bash
# 前提：设置 API Key
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

# 运行测试
go test ./...
```

---

## 架构

```
用户文本
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent → 风格/情绪/特征向量
   └── PlanSong    → 和弦/调性/BPM
   ↓
SongComposer (Go 规则引擎, 0 次 LLM)
   ├── EmotionEngine    → 情绪→音乐参数
   ├── Motif Engine     → 动机展开 (A-A'-B-A)
   ├── Drums Generator  → 能量驱动鼓模式
   ├── Bass Generator   → 根音/五音
   └── Pad Generator    → 和弦铺垫
   ↓
Post-processing
   ├── Transition Engine   → 段落过渡
   ├── Dynamic Layering    → 乐器数量控制
   ├── Texture Layer       → 氛围铺底
   ├── Creative Chaos      → 故意犯错
   ├── MelodyGrammar       → 调性/音程约束
   └── MusicDNA Extractor  → 结构化分析
   ↓
MIDI → .mid 文件 + Stem 分轨
```

**核心原则**: LLM 只做高层决策（风格、情绪、和弦），所有编曲由 Go 规则引擎确定性执行。

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
│   │   │   ├── song.go           Song Composer (完整编曲)
│   │   │   ├── motif_engine.go   Motif Engine (变奏/展开)
│   │   │   ├── motif.go          动机发展 + 副旋律
│   │   │   ├── melody.go         MelodyGrammar (调性约束)
│   │   │   ├── transition.go     Transition Engine (段落过渡)
│   │   │   ├── dynamics.go       Dynamic Layering (乐器数量)
│   │   │   ├── texture.go        Texture Layer (氛围铺底)
│   │   │   ├── groove.go         Swing 量化 + 乐句呼吸
│   │   │   ├── energy.go         能量曲线 + Bass-Kick 对齐
│   │   │   ├── structure.go      段落模板
│   │   │   ├── stems.go          Stem 分轨导出
│   │   │   ├── substitution.go   和弦代换
│   │   │   └── dna.go            ComposerDNA 人格 + SongMemory
│   │   ├── musicdna/
│   │   │   ├── types.go          MusicDNA 数据结构
│   │   │   └── extractor.go      MIDI→DNA 提取器
│   │   ├── generator/            规则乐器生成器
│   │   ├── harmony/              和声约束
│   │   ├── llm/                  Prompt 模板
│   │   ├── midi/                 原生 SMF Type 1 写入
│   │   ├── music/                乐理工具
│   │   ├── mutation/             Creative Chaos
│   │   ├── schema/               数据类型
│   │   └── style/                风格数据库 + 配器模板
│   ├── 编曲方法/                  编曲知识库
│   └── midi_output/              已生成 MIDI
│
└── frontend/index.html           Web 界面
```

---

## 核心系统

### 1. Emotion Engine (情绪引擎)

将自然语言情绪映射为 5 维音乐参数：

| 维度 | 范围 | 作用 |
|------|------|------|
| **Tension** | 0-1 | 紧张度 → 和弦复杂度 (0=三和弦, 1=七/增/减) |
| **Energy** | 0-1 | 能量 → 节奏密度 (0=稀疏, 1=密集) |
| **Warmth** | 0-1 | 温暖度 → 音色选择 (0=冷合成器, 1=钢琴/弦乐) |
| **Stability** | 0-1 | 稳定度 → 动机重复率 (0=不断变奏, 1=重复) |
| **Brightness** | 0-1 | 明亮度 → 音区控制 (0=低八度, 1=高八度) |

### 2. Motif Engine (动机引擎)

用短动机 (3-8 音) 通过 **A-A'-B-A** 结构展开全曲旋律：

- **Transpose**: 平移 (±2/±5 半音)
- **Invert**: 镜像反转 (+2+3 → -2-3)
- **Retrograde**: 逆行 (C-D-E → E-D-C)
- **Fragment**: 截断 (只用前 2-3 音)
- **Extend**: 延展 (追加经过音)

### 3. Song Composer (作曲系统)

从动机 + 和弦 + 情绪展开完整多轨编曲：

```
Timeline Planner → 段落结构
Motif Allocation → 每段用哪种动机变体
Drums Generator → 能量驱动鼓模式
Bass Generator  → 根音/五音
Pad Generator   → 和弦铺垫
```

### 4. MusicDNA (结构化音乐分析)

每首 MIDI 输出结构化分析报告：

```
===== MusicDNA =====
--- Structure ---
  intro: bars 0-1, energy=0.44
  verse: bars 2-3, energy=0.51
  chorus: bars 4-5, energy=0.47
--- Harmony ---
  Key: C major
  bar 0: C
  bar 1: G
  bar 2: Am
--- Motif ---
  Pattern (intervals): [0 2 4 3 0]
  Confidence: 0.83
  Variants: transpose, invert
```

### 5. ComposerDNA (人格系统)

8 种内置作曲家人格，影响动机重复率、混沌程度、声部交叉规则：

| 人格 | MotifObsession | Chaos | 适用场景 |
|------|----------------|-------|---------|
| **Toby Fox (Undertale)** | 0.90 | 0.30 | 游戏音乐 |
| **Nobuo Uematsu (FF)** | 0.70 | 0.10 | JRPG |
| **Hans Zimmer** | 0.85 | 0.20 | 史诗/电影 |
| **Hyperpop Maniac** | 0.40 | 0.90 | 实验 |
| **Classical Purist** | 0.60 | 0.05 | 古典 |

---

## 生成流程 (15 阶段)

| 阶段 | 模块 | 说明 |
|------|------|------|
| 1 | ParseIntent | LLM 解析文本 → 风格/情绪/特征向量 |
| 2 | PlanSong | LLM 生成和弦 + 调性 + BPM |
| 3 | EmotionEngine | 情绪 → Tension/Energy/Warmth/Stability/Brightness |
| 4 | Timeline Planner | 段落结构 (intro/verse/chorus/bridge/outro) |
| 5 | Motif Engine | 动机变奏展开 (A-A'-B-A) |
| 6 | Drums Generator | 能量驱动鼓模式 |
| 7 | Bass Generator | 根音/五音贝斯线 |
| 8 | Pad Generator | 和弦铺底 |
| 9 | Transition Engine | 段落过渡 (6 种类型) |
| 10 | Dynamic Layering | 乐器数量控制 |
| 11 | Texture Layer | 氛围铺底 |
| 12 | MelodyGrammar | 调性/音程几何约束 |
| 13 | Creative Chaos | 故意犯错 |
| 14 | MusicDNA Extractor | 结构化分析 |
| 15 | MIDI Render | .mid 文件 |

---

## 生成示例

`midi_output/` 目录下的所有 MIDI 可直接拖入 DAW (FL Studio / Ableton / Logic / Cubase)。

| 文件 | 风格 | 音轨 | 音符数 |
|------|------|------|--------|
| 晴天风 流行钢琴.mid | 周杰伦风格 | 4 | 148 |
| RPG Battle Charge.mid | 史诗战斗 | 6 | 276 |
| 深夜emo.mid | 忧郁抒情 | 1 | 48 |
| Final Fantasy Battle.mid | JRPG 战斗 | 6 | 266 |
| Sunny Day Memory.mid | 流行钢琴 | 4 | 148 |

---

## 编曲知识库

项目内置三份专业编曲知识库 (`backend/编曲方法/`)：

- **liuxing.md** — 流行歌编曲
- **rap.md** — 说唱编曲
- **rock.md** — 摇滚编曲

这些知识被注入到 LLM Prompt 中指导生成。

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生写入, 零外部依赖
- **前端**: 纯 HTML/JS (无框架)

## License

MIT
