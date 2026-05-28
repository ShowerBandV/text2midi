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
LLM Agent (3 次调用)
   ├── ParseIntent       → 风格/情绪/特征向量
   ├── PlanSong          → 和弦/调性/BPM
   └── GenerateMelody    → LLM 直接写主旋律 (不经过 Go 引擎)
   ↓
作曲引擎 (Go 规则引擎)
   ├── Planner     → 结构化 SongPlan（段落/能量/配器/tempo/拍号）
   ├── Composer    → 多轨编曲（bass/drums/pad/fx — 不含 lead）
   ├── Arranger    → 编曲协调（skip lead 音轨）
   ├── Critic      → 质量评估（skip lead 音轨）
   ├── Humanizer   → 人性化（velocity/timing drift）
   └── MusicDNA    → 结构化分析报告
   ↓
Post-processing (仅对伴奏轨)
   ├── Transition Engine → 段落过渡
   ├── Dynamic Layering  → 乐器数量控制
   └── Texture Layer     → 氛围铺底
   ↓
MIDI → .mid 文件 + MusicDNA 分析
```

---

## V2 架构详解

### 1. Planner（歌曲规划器）

输入特征向量，输出结构化 SongPlan，每段独立参数：

```
SongPlan:
  intro:  4 bars, energy=0.2, tempo=120, ts=4/4
  verse:  4 bars, energy=0.4, tempo=120, ts=4/4
  pre:    2 bars, energy=0.6, tempo=120, ts=4/4
  chorus: 4 bars, energy=0.85, tempo=120, ts=4/4
  bridge: 2 bars, energy=0.5, tempo=75,  ts=4/4   ← 渐慢
  outro:  2 bars, energy=0.15, tempo=65, ts=4/4   ← 更慢
```

风格感知的段落布局：

| 风格 | 触发条件 | 段落结构 |
|------|---------|---------|
| **Metal** | dark>0.7, energy>0.7 | intro(1) → verse(2) → chorus(4) → bridge(1, slower) |
| **Pop** | energy>0.4, rhythmic<0.5 | intro(2) → verse(4) → pre(2) → chorus(4) → bridge(2, 75bpm) → outro(2, 65bpm) |
| **Hip-hop** | rhythmic>0.5, energy>0.3 | intro(1) → loop_a(3) → loop_b(3) → outro(1) |
| **Ambient** | default | intro(2) → verse(4) → chorus(4) → outro(2) |

### 2. Motif Engine（动机引擎）

5 种变奏方法 + 全局注册表：

| 变奏 | 方法 | 示例 |
|------|------|------|
| Transpose | 平移 ±N 半音 | C-D-E → D-E-F |
| Invert | 镜像反转 | +2+3 → -2-3 |
| Retrograde | 逆行 | C-D-E → E-D-C |
| Fragment | 截断 | 保留前 N 音 |
| Extend | 延展 | 追加经过音 |

### 3. Phrase（乐句系统）

4 小节 = 1 乐句，question (bars 1-2) + answer (bars 3-4)

| 风格 | Bar 0 | Bar 1 | Bar 2 | Bar 3 |
|------|-------|-------|-------|-------|
| **Metal** | Riff | 八度下移 | Riff 重复 | 碎片+变奏 |
| **Pop** | A (motif) | A' (+3 移调) | B (对比) | A (回归) |
| **Hip-hop** | Loop | Loop | Loop | Loop (+3 变奏) |
| **Ambient** | 2 音 | +2 移调 | +4 移调 | +6 移调 |

### 4. Arranger（编曲协调器）

检测并修复 4 种冲突：

| 冲突类型 | 检测条件 | 修复方法 |
|---------|---------|---------|
| **音区冲突** | bass 和 lead 在同一中频区 | 贝斯降八度 |
| **密度过载** | 单小节总音符 > 32 | 降低部分轨道力度 |
| **空小节** | 没有任何音符 | 自动填充 |
| **节奏冲突** | 鼓密集 + 旋律也密集 | 旋律简化 |

### 5. Critic（质量评估器）

5 维评分 + 自动修复：

```
Score = repetition*0.25 + tension*0.2 + groove*0.2 + climax*0.2 + density*0.15
```

`if score.Climax < 0.5 → 后 1/3 能量提升 20%`

### 6. Humanizer（人性化）

- **Velocity Drift**: 每音符力度 ± 随机偏移
- **Timing Drift**: 每音符起始时间 ± 随机偏移
- **Ghost Notes**: 弱拍随机添加极轻的幽灵音
- **Accent Beat**: 每小节第一拍加重

### 7. MusicDNA（结构化分析）

每首 MIDI 输出完整报告：

```
===== MusicDNA =====
--- Structure ---   intro/verse/chorus, energy, density
--- Harmony ---     Key + 每小节和弦
--- Motif ---       动机 pattern + 四项评分（repetition/contour/simplicity/rhythm）
--- Rhythm ---      density/swing/syncopation
--- Texture ---     每轨音域/角色/音符数
--- Dynamics ---    力度范围/平均值
--- Emotion ---     tension/energy/warmth/stability/brightness
```

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 生成器
│   ├── internal/
│   │   ├── agent/                LLM Agent
│   │   ├── planner/              V2: 歌曲规划器（tempo/ts 感知）
│   │   ├── phrase/               V2: 乐句系统
│   │   ├── arranger/             V2: 编曲协调器
│   │   ├── critic/               V2: 质量评估器
│   │   ├── motif/                V2: 动机注册表
│   │   ├── composer/             作曲引擎（10+ 模块）
│   │   │   ├── song.go           Song Composer
│   │   │   ├── emotion.go        Emotion Engine
│   │   │   ├── humanizer.go      Humanizer
│   │   │   ├── phrase.go         风格感知句式
│   │   │   ├── transition.go     Transition Engine
│   │   │   ├── dynamics.go       Dynamic Layering
│   │   │   ├── energy.go         能量曲线
│   │   │   ├── texture.go        Texture Layer
│   │   │   ├── groove.go         Swing 量化
│   │   │   ├── stems.go          Stem 导出
│   │   │   ├── melody.go         MelodyGrammar
│   │   │   ├── motif_engine.go   Motif Engine
│   │   │   └── dna.go            ComposerDNA
│   │   ├── musicdna/             结构化音乐分析
│   │   ├── generator/            规则乐器生成器
│   │   ├── llm/                  Prompt 模板
│   │   ├── midi/                 原生 SMF Type 1
│   │   ├── music/                乐理工具
│   │   ├── mutation/             Creative Chaos
│   │   ├── schema/               数据类型
│   │   └── style/                风格数据库
│   ├── 编曲方法/                  编曲知识库
│   └── midi_output/              已生成 MIDI
│
├── frontend/index.html           Web 界面
├── ROADMAP.md                    路线图
├── HARDCODED.md                  硬编码审计
└── TODO.md                       当前待办
```

---

## 完整生成流程 (17 阶段)

| 阶段 | 模块 | 说明 |
|------|------|------|
| 1 | ParseIntent | LLM 解析文本 → 风格/情绪/特征向量 |
| 2 | PlanSong | LLM 生成和弦 + 调性 + BPM |
| 3 | **Planner** | 结构化 SongPlan（段落/能量/tempo/拍号） |
| 4 | **Motif** | 动机注册 + 变体生成 |
| 5 | **Phrase** | 4-bar 乐句展开（按风格选句式） |
| 6 | Composer | 多轨编曲（bass/drums/lead/pad/fx） |
| 7 | Humanizer | velocity/timing drift + ghost note |
| 8 | **Arranger** | 编曲协调（冲突检测 + 修复） |
| 9 | **Critic** | 质量评估 + 自动修复 |
| 10 | MusicDNA | 结构化分析报告 |
| 11 | MelodyGrammar | 调性/音程约束 |
| 12 | Transition Engine | 段落过渡 |
| 13 | Dynamic Layering | 乐器数量控制 |
| 14 | Texture Layer | 氛围铺底 |
| 15 | Creative Chaos | 故意犯错 |
| 16 | MIDI Render | .mid 文件 |

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生写入，零外部依赖
- **前端**: 纯 HTML/JS

## License

MIT
