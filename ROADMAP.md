# text2midi 路线图

## 第二代架构（V2）

### 核心转变：从"生成器"到"DAW Copilot"

```
当前:  一次性生成 MIDI → 不可编辑
目标:  结构化生成 → 局部重生成 → 风格迁移 → 轨道替换
```

### V2 架构

```
internal/
├── planner/        歌曲规划器（LLM 输出结构化 plan）
│   ├── plan.go         SongPlan / SectionPlan
│   ├── timeline.go     时间线布局
│   └── template.go     段落模板匹配
│
├── phrase/         乐句系统（按 phrase 而不是 bar 生成）
│   ├── phrase.go       Phrase / 4-bar question-answer
│   └── builder.go      从 section → phrase → bar → notes
│
├── motif/          主题记忆（跨段落重复 + 变奏）
│   ├── motif.go        Motif / 5 种变奏方法
│   └── registry.go     全局 motif 注册 + 检索
│
├── arranger/       编曲协调器（空间管理）
│   ├── arranger.go     ArrangementState / 音区冲突检测
│   ├── density.go      节奏密度调节
│   └── register.go     自动留白 / 轨道协调
│
├── critic/         批判器（生成后质量评估）
│   ├── critic.go       MusicScore / 四项评分
│   └── repair.go       低分触发局部重生成
│
├── performance/    演奏层（人性化）
│   ├── humanize.go     timing drift / velocity cluster / ghost notes
│   └── groove.go       laid back / swing 局部化
│
├── retrieval/      RAG for Music
│   ├── pattern.go      PatternEmbedding / 和弦-节奏-风格索引
│   └── search.go       DNA 模板相似度检索
│
├── composer/        总控（已有，需重构为分段生成）
│   ├── song.go         接收 planner → 分派给 arranger/generator
│   └── context.go      GenerationContext / DNA state
│
├── generator/       规则生成器（已有）
│   ├── bass.go
│   ├── drum.go
│   ├── chord.go
│   ├── lead.go
│   └── fx.go
│
└── musicdna/        结构化音乐分析（已有）
    ├── types.go
    └── extractor.go
```

### 10 个核心模块详解

#### 1. planner（规划器）
- 输入：LLM 解析结果（style / mood / feature_vector）
- 输出：SongPlan { BPM, Key, Sections []SectionPlan }
- 每个 SectionPlan：{ Name, Bars, Energy, Density, Instruments, MotifIDs }
- **为什么重要**：音乐是"结构优先"，不是 note 优先

#### 2. phrase（乐句系统）
- 4-bar question + 4-bar answer 结构
- Phrase { Bars, Tension, Resolution, MotifID }
- 生成流程：section → phrase → bar → notes
- **为什么重要**：人类按 phrase 作曲，按 bar 生成是 AI 味的根源

#### 3. motif（主题记忆）
- 跨段落跟踪 motif
- 第一次副歌 → Motif A，第二次副歌 → Motif A'
- 变奏：Transpose / Invert / RhythmMutation / OctaveShift
- **为什么重要**：没有 recurring motif 就不像歌

#### 4. arranger（编曲协调器）
- ArrangementState { Density, RegisterUsage, RhythmComplexity }
- 检测音区冲突（bass 和 lead 不要同时中频）
- 节奏冲突时自动调整（鼓密→旋律简化）
- 自动留白（chorus 前空一拍）
- **为什么重要**：这是"高级感"的来源

#### 5. critic（批判器）
- MusicScore { Repetition, Tension, Groove, Climax }
- 检测：melody collapse / rhythm collapse / empty arrangement / no climax
- `if score.Climax < 0.5 { regenerateChorus() }`
- **为什么重要**：质量飞跃的关键闭环

#### 6. performance（演奏层）
- timing drift（± 随机毫秒偏移）
- velocity cluster（句子级动态，不是随机）
- ghost notes（特别是鼓）
- laid back groove（snare 略靠后）
- **为什么重要**：让 MIDI 从"机械"变"真人"

#### 7. retrieval（RAG for Music）
- PatternEmbedding { Chords, Rhythm, Mood }
- 检索：chord progression / rhythm pattern / bass groove / arrangement style
- **为什么重要**：纯生成不稳定，检索 + 条件生成才是工程级方案

#### 8. DNA 系统升级
- DNA 不再是 metadata，而是直接参与采样
- `dna.ApplyToMelody()` / `dna.ApplyToRhythm()` / `dna.ApplyToHarmony()`
- 周杰伦 DNA → 偏好大跳后回归
- City Pop DNA → 偏好 maj7
- **为什么重要**：DNA 影响每次 note 决策

#### 9. Style 系统重构
- 从离散 enum 改为连续 StyleVector
- `type StyleVector struct { Warmth, Aggressive, Acoustic, Groove, Complexity }`
- "cyberpunk jazz" → 0.6 cyberpunk + 0.4 jazz
- **为什么重要**：风格无限组合，不再固定

#### 10. DAW Copilot 能力
- 局部重生成（第 9-16 小节）
- 风格迁移（保持 melody 换 citypop）
- 局部编辑（"让副歌更燃"）
- 轨道替换（重做 bass）
- **为什么重要**：MIDI-first 系统的真正优势，和 Suno 的根本区别

### 3 个月执行计划

| 月份 | 模块 | 产出 |
|------|------|------|
| **第 1 月** | planner + phrase + motif | 结构化规划 + 乐句系统 + 主题记忆 |
| **第 2 月** | arranger + critic + performance | 编曲协调 + 质量评估 + 人性化 |
| **第 3 月** | retrieval + DAW edit API + partial regen | RAG + 局部编辑 + 风格迁移 |

## Phase 1: DNA 系统打地基（当前）

### Structure Extractor（音乐分镜）
- [ ] BarFeature 计算（density / energy / chord / instrument_count）
- [ ] 变化点检测（energy / density / chord 阈值）
- [ ] 段落聚类（similarity-based section grouping）
- [ ] 模板匹配修正（AABA / ABAB / intro-verse-chorus）
- [ ] 输出 StructureDNA

### Motif Scoring System（旋律质量评估）
- [ ] MotifScore { Repetition, Contour, Simplicity, RhythmIdentity }
- [ ] Repetition 计算（occurrence / total bars）
- [ ] Contour 计算（slope variance）
- [ ] Simplicity 计算（avg interval size）
- [ ] RhythmIdentity 计算（duration pattern similarity）
- [ ] 总分 = repetition*0.4 + contour*0.2 + simplicity*0.2 + rhythm*0.2
- [ ] 评分驱动生成（高分→多重复，低分→强变异）

### MIDI → MusicDNA Extractor
- [ ] StructureDNA（段落提取）
- [ ] HarmonyDNA（和弦检测）
- [ ] MotifDNA（动机提取 + 评分）
- [ ] RhythmDNA（节奏模式）
- [ ] TextureDNA（编曲层）
- [ ] MIDI Cleaner（过滤噪音 MIDI）
- [ ] MusicDNA Schema 固化

## Phase 2: DNA 资产库

- [ ] DNA Library（.dna 模板文件）
- [ ] DNA 评分系统（质量过滤）
- [ ] 按风格分类收集 MIDI

## Phase 3: 生成系统

- [ ] Motif Engine（repeat / variation / fragment / invert）
- [ ] Harmony Engine（和弦生成 + tension 控制）
- [ ] Rhythm Engine（swing / syncopation / density）
- [ ] Structure Composer（intro → verse → chorus）

## Phase 4: 情绪系统

- [ ] EmotionDNA（tension / energy / stability / brightness）
- [ ] Emotion → Music Mapping

## Phase 5: 系统合成

- [ ] Song Composer 总控（已初步完成）
- [ ] Multi-track Engine（drums / bass / lead / pad / fx）

## Phase 6: 进阶能力

- [ ] DNA ↔ MIDI 双向系统
- [ ] Style Template System
- [ ] Motif Reuse / Cross-song

## Phase 7: 产品化

- [ ] DSL 输入系统
- [ ] API（/generate / /dna/extract / /dna/mutate）
- [ ] Web UI

## 架构诊断

### 当前瓶颈
composer 层还只是"参数映射器"，不是真正的"作曲决策器"。

### 与纯音频 AI 的差异化
不是"AI 生成音乐"，而是"AI 辅助作曲 DAW"——生成的是结构化、可编辑、可局部修改的音乐。
