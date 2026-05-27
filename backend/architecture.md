# 程序化作曲引擎 — 架构设计

当前定位：Adaptive Game Music Engine（自适应游戏音乐引擎）
放弃通用歌曲生成，专注 Loop-aware、程序化、可交互的游戏音乐。

---

## 核心理念

```
LLM 高层决策 + Go 规则展开 = 可控、稳定、有风格的作曲系统
```

| 层 | 职责 | 技术 |
|----|------|------|
| **LLM** | 风格、结构、Motif Seed、情绪 | DeepSeek / GPT |
| **Go Core** | 编曲展开、和声、节奏、层次、动态 | 规则引擎 + DNA |
| **Pattern Lib** | 预置 Groove / Fill / Arp | 数据库 + 组合 |
| **Adapter** | Wwise / FMOD / Stem 导出 | 中间件 |

---

## 系统架构

```
用户输入 (prompt / game state)
    │
    ▼
┌─────────────────────────────────────┐
│         LLM 决策层 (轻量)            │
│                                     │
│  ParseIntent    → 风格, 情绪         │
│  PlanStructure  → 段落结构, 时长      │
│  SeedMotif      → 3-5 音动机种子    │
│                                     │
│  总共 2-3 次 LLM 调用 (减半)         │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│      Go 展开层 (规则引擎)             │
│                                     │
│  SongMemory        ← 全曲作曲逻辑    │
│  ArrangementTL     ← 时间线驱动      │
│  MotifDevelopment  ← 变奏 + 回归     │
│  HarmonyEngine     ← 和声 + 代换     │
│  RhythmEngine      ← Groove + Fill   │
│  DynamicControl    ← 层次 + 密度     │
│  TextureLayer      ← 氛围铺底        │
│  TransitionEngine  ← 段落过渡        │
│  StemExporter      ← 分轨导出        │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│       Pattern Library (数据库)        │
│                                     │
│  drum_grooves/     ← 50+ 鼓模式     │
│  bass_lines/       ← 30+ 贝斯线     │
│  arp_patterns/     ← 20+ 琶音       │
│  transitions/      ← 20+ 过渡填充    │
│  atmosphere/       ← 10+ 氛围铺底    │
│                                     │
│  LLM 组合 → Go 执行 → 不是从头生成   │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│    渲染层                             │
│                                     │
│  MIDI Render       ← .mid           │
│  Stem Export       ← 分轨导出        │
│  Wwise Adapter     ← Game Audio SDK │
│  Adaptive Stream   ← 实时响应状态     │
└─────────────────────────────────────┘
```

---

## 核心模块设计

### 1. SongMemory — 全曲作曲逻辑

当前 `SongDNA` 只是一个"动机缓存"。

需要的升级：

```go
type SongMemory struct {
    MotifSeed          []int        // LLM 生成的 3-5 音动机
    MotifVariants      []MotifVar   // 展开后的所有变奏
    RhythmSignature    []float64    // 节奏标识 (♪ ♩ ♪♪)
    HarmonicIdentity   []int        // 和弦倾向
    EnergyTimeline     []float64    // 每小节能量目标
    InstrumentTimeline []string     // 每小节活跃乐器
    EmotionArc         string       // 情绪曲线
    CallbackSchedule   []Callback   // 回调计划
}
```

设计原则：**所有后处理步骤都引用 SongMemory，而不是独立决策。**

### 2. Arrangement Timeline — 时间线驱动

当前的生成是**轨道先存在，再处理**。改成**时间线先定，再填充**。

```go
type Section struct {
    Name               string   // "intro" / "verse" / "chorus" / "bridge"
    Bars               int
    Energy             float64  // 0.0-1.0
    ActiveInstruments  []string // 当前段落哪些乐器在演奏
    MotifVariant       int      // 使用第几种动机变奏
    DrumPattern        string   // 鼓模式名称
    HarmonyDensity     int      // 和弦密度 (3音/4音/5音)
    TextureType        string   // 氛围类型 (drone/pad/noise/none)
    TransitionOut      string   // 段落结束时的过渡类型
}

type ArrangementTimeline struct {
    Sections []Section
    BPM      int
    Key      string
}
```

### 3. Motif Development Engine

从"局部变奏"升级为"主题回归"。

```
Verse (A):     Motif A   → 原始动机
Chorus (A'):   Motif A   → 八度上移 + 节奏加密
Bridge (B):    Motif A   → 倒影 (上下颠倒)
Final Chorus:  Motif A   → 原始 + 副旋律
Outro:         Motif A   → 慢速、弱化
```

实现方式：`SongMemory.MotifVariants` 存储所有变奏，`Section.MotifVariant` 引用。

### 4. Transition Engine — 段落过渡

自动在段落之间插入过渡。

```go
type Transition struct {
    Type      string // "drum_fill" / "riser" / "cymbal_reverse" / "silence" / "bass_slide"
    Duration  int    // ticks
    Intensity float64
}

// GetTransition 根据前后段落的能量差自动选择过渡。
func GetTransition(fromEnergy, toEnergy float64) *Transition {
    if toEnergy > fromEnergy + 0.3 {
        return &Transition{Type: "riser", Duration: 8, Intensity: 0.8}
    }
    if toEnergy < fromEnergy - 0.3 {
        return &Transition{Type: "silence", Duration: 4, Intensity: 0.3}
    }
    return &Transition{Type: "drum_fill", Duration: 4, Intensity: 0.5}
}
```

### 5. Dynamic Control — 真·动态系统

控制**乐器数量**比控制 `velocity` 更重要。

```go
type DynamicLayer struct {
    EnergyThreshold float64
    Instruments     []string
}
var DynamicLayers = []DynamicLayer{
    {EnergyThreshold: 0.0, Instruments: []string{"pad"}},
    {EnergyThreshold: 0.3, Instruments: []string{"pad", "bass", "hihat"}},
    {EnergyThreshold: 0.6, Instruments: []string{"pad", "bass", "hihat", "snare", "chords"}},
    {EnergyThreshold: 0.8, Instruments: []string{"pad", "bass", "drums_full", "chords", "lead", "counter"}},
}
```

### 6. Texture Layer — 氛围铺底

游戏音乐特别需要的音层。

```go
type TextureType int
const (
    TextureNone     TextureType = iota
    TexturePad      // 长音和弦铺垫
    TextureDrone    // 持续单音嗡鸣
    TextureNoise    // 噪音/氛围
    TextureArp      // 快速琶音
    TextureReverse  // 反向钢琴
)
```

### 7. Rhythm Identity — 节奏动机

SongMemory 记录 RhythmSignature，跨段落复用。

```go
type RhythmCell struct {
    Durations []float64 // 相对时值
    Accents   []bool    // 重音位置
}
```

### 8. Stem Exporter — 分轨导出

```go
// ExportStems generates separate .mid files per instrument group.
func ExportStems(midiIR schema.MidiIR, outputDir string) error {
    // drums.mid, bass.mid, lead.mid, pads.mid, texture.mid
}
```

### 9. Pattern Library — 组合而非生成

```go
type Pattern struct {
    Name     string
    Category string // "drum_groove" / "bass_line" / "arp"
    Tags     []string
    Data     []byte  // MIDI bar snippet
}

var patternDB []Pattern // 预置 50+ 模式

// ComposeFromPattern 用 LLM 选择模式，Go 组合。
func ComposeFromPattern(patterns []Pattern, timeline ArrangementTimeline) MidiIR {
    // 选择模式 → 组合到时间线 → 输出
}
```

---

## LLM 调用次数减半

### 当前 (6 次)

```
ParseIntent → PlanSong → PlanArrangement → GeneratePatterns → GenerateMelody → GenerateBass
```

### 目标 (2-3 次)

```
LLM 1: ParseIntent + PlanStructure + SeedMotif  (合并)
LLM 2: SelectPatterns (从 Pattern Library 选择)
```

Go 规则引擎展开所有编曲细节。

---

## 输出格式

### MIDI (当前)

```
.mid 文件
```

### Stem (新增)

```
project/
├── drums.mid
├── bass.mid
├── lead.mid
├── pads.mid
├── texture.mid
└── full.mix.mid
```

### Wwise / FMOD Adapter (远期)

```
Stem → Wwise SoundBank → Unity/Unreal
Stem → FMOD Event → Godot/Unity
```

---

## 优先级路线图

| 阶段 | 模块 | 预计 |
|------|------|------|
| **P0** | Transition Engine + 段落过渡 | 2 天 |
| **P0** | 真·动态控制 (乐器数量) | 1 天 |
| **P1** | SongMemory 升级 (全曲作曲逻辑) | 2 天 |
| **P1** | 主题回归系统 | 1 天 |
| **P2** | Rhythm Identity | 1 天 |
| **P2** | Texture Layer | 1 天 |
| **P3** | Pattern Library | 3 天 |
| **P3** | Stem Exporter | 1 天 |
| **P4** | Wwise / FMOD Adapter | 5 天 |
| **P4** | Adaptive Stream (Game State Input) | 5 天 |
