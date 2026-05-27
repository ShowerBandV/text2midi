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

## 当前优先级

1. Structure Extractor（段落拆解）
2. Motif 评分系统（好旋律的判断标准）
3. DNA Library
4. MIDI Cleaner
