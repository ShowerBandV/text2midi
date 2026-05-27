# text2midi 路线图

## Phase 1: DNA 系统打地基（当前）

- [ ] MIDI → MusicDNA Extractor 完成
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
