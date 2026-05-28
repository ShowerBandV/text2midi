# text2midi 路线图

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

## 架构诊断（2025.06）

### 当前瓶颈

composer 层还只是"参数映射器"，不是真正的"作曲决策器"。

### 缺失的关键能力

1. **Section-aware Composer** — 每个段落 (intro/verse/pre/chorus/bridge/outro) 分别生成，而非整曲统一处理
2. **Motif Memory** — 主题跨段落记忆 + 回归，不是每小节重新生成
3. **Iterative Generation** — 多轮协同生成 (chord→bass→melody→drum→回头修 melody)
4. **Style Vector** — 从离散的 style enum 改为连续的 style embedding (warmth/density/acoustic/groove/complexity)
5. **Humanizer** — velocity drift / timing drift / ghost note / imperfect quantize / 局部 swing
6. **Planner 层** — 作曲 Agent 应输出结构化 plan (tempo/key/sections/motifs)，而非单维 genre tag

### 下一阶段建议新增目录

```
internal/
├── planner/       作曲规划器（LLM 输出结构化 plan）
├── motif/         主题记忆 + 回归系统
├── arranger/      编曲层（section-aware 编排）
├── humanizer/     人性化（velocity drift / timing drift / ghost note）
├── retrieval/     DNA 模板检索
└── memory/        跨曲目记忆
```

### 差异化优势

不是"AI 生成音乐"，而是"AI 辅助作曲 DAW"——生成的是结构化音乐，可编辑、可局部修改、可保留和弦换旋律。这是和 Suno 等纯音频 AI 的根本区别。

## 当前优先级

1. Section-aware Composer（段落分别生成）
2. Motif Memory（主题记忆 + 回归）
3. Structure Extractor（段落拆解）
4. Motif Scoring System
