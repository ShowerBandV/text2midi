# Roadmap

## 已完成 ✅

### 工程基础
- [x] LLM + Local 双模式
- [x] 8 种风格支持（rpg / emo / rock / punk / metal / pop / trap / ambient）
- [x] 并行 LLM 生成（goroutine + WaitGroup）
- [x] Token 实时追踪 + USD/CNY 成本显示
- [x] `--dry-run` 先看 plan 再生成
- [x] `--flat-vel` 统一力度
- [x] `--pentatonic` 五声音阶 + 中国风装饰
- [x] `--refine` LLM 主观评审迭代
- [x] 渲染前验证（validateMidiIR）
- [x] Stage checkpoint（intent / plan / arrangement JSON）
- [x] 和弦知识外部化（knowledges/chords.md）

### 歌曲结构
- [x] 标准歌曲结构：Intro → Verse → Chorus → Bridge → Chorus → Outro
- [x] `songSection()` 自动段落划分
- [x] 最小 2 分钟歌曲长度（各风格默认 bars 适配）

### 鼓
- [x] Punk：D-beat + ghost drag + flam + crash-ride + tom fill + 开镲副歌
- [x] Metal：双踩 16th + china + ride bell + 反拍 kick + descending tom fill
- [x] Rock：Backbeat + off-beat push + crash + tom/kick-snare fill
- [x] Pop：Swing hi-hat + rim-click verse + full snare chorus + 加速 fill
- [x] Trap：Sparse kick + 16th hi-hat triplet + 32nd roll + clap + open hat punctuation
- [x] Emo：Sparse rim-click + floor tom fill

### Bass
- [x] Punk：直八分音符根音，简单暴力
- [x] Metal：8th chug + 16th fill，锁死 double-kick
- [x] Pop：旋律化 chord tones（root-5th-octave-3rd），verse 稀疏 chorus 密集
- [x] Trap：808 sub-bass C1，长 sustain，pitch slide
- [x] Emo：长音 sustain + 八度跃

### 和弦
- [x] Punk/Metal/Rock：强力和弦（根+五），C1 降调
- [x] Pop/RPG：Maj7/9th 丰富和声 voicing
- [x] Trap：暗色 minimal pad（根+五+八），vel 45
- [x] Ambient：开放排列 wide interval

### 主音 / 旋律
- [x] Metal：Bodom 风格 riff→arpeggio→solo→silence，harmonic minor + sweep/tap + tremolo + whammy dive
- [x] Pop/RPG：John Legend 风格左手八度+右手和弦+旋律碎片，段落分明
- [x] Punk：简单 3-4 音短 lick + 八度双音
- [x] Twin harmony：Metal 平行三度/八度齐奏/反向进行
- [x] Counter-melody：独立节奏型 + 延迟进入
- [x] 旋律拱形结构（shapeMelodyArch）
- [x] MelodyGrammar 后处理（音阶约束 + 音程限制 + 重力回归）

### 配器 / 编制
- [x] 风格感知 track layout（punk 四件套 / emo 钢琴弦乐 / trap 808+合成器）
- [x] 层次弦乐（Strings-Layered）：intro 单音 → pre 双音 → chorus 全编制

---

## 待完成 🔲

### P0 — 工程基础

- [ ] **Stage checkpoint 完整化**：把 track_events 也存入 stage JSON，`--resume` 可从任意 stage 恢复
- [ ] **Local 模式并行生成**：鼓/bass/和弦/主音四轨 goroutine 并行（目前 LLM 模式已并行，local 还是串行）
- [ ] **Rock / Punk lead 专用生成器**：Rock = blues-scale double-stop + 重复 lick；Punk = 八度双音短 lick
- [ ] **各风格默认 bars 适配 2 分钟**：目前只改了 metal/pop/rpg，rock/punk/emo/trap/ambient 还是 24-32 bars

### P1 — 质量保障

- [ ] **渲染前验证加强**：小节完整性检查、音域范围、声部对齐（参考 Clef 的 music21 validate_abc.py）
- [ ] **Riff 性格化**：给 rock/punk 的 riff 增加 repeating hook + variation answer 结构
- [ ] **`--refine` 迭代 Snapshot 回滚**：每次迭代前自动备份 evMap，迭代后可回滚（参考 Clef 的 Snapshot 版本管理）
- [ ] **Metal 节奏吉他 gallop 细化**：目前每 2 小节切换标准/gallop，应该按段落控制密度

### P2 — 功能扩展

- [ ] **SF2 profile 支持**：加载 SF2 的 key_range/sweet_spot，约束生成音域（参考 Clef 的 SF2 感知）
- [ ] **音频渲染**：MIDI → WAV → MP3（集成 FluidSynth 或 Go 音频库，参考 Midra）
- [ ] **Rock lead blues-scale**：推弦感、double-stop、重复 lick，区别于 pop 的干净旋律
- [ ] **Metal neo-classical 恢复**：diminished run + 半音经过音 + 模进（目前 bridge 的 solo 只有 sweep/tremolo/whammy）
- [ ] **Punk 不需要主音吉他时自动跳过**：只在 intro/chorus 出现短 lick，其余时候休息

### P3 — 远期

- [ ] **ABC 记谱输出**：可选输出 ABC 格式的乐谱，方便人类审查和修改（参考 Clef）
- [ ] **Web 前端**：React SPA + API 服务（参考 Midra 的 FastAPI + React）
- [ ] **实时试听**：浏览器内 MIDI 播放器
- [ ] **更多风格**：jazz / blues / funk / reggae / EDM
- [ ] **和弦模板扩展**：knowledges/chords.md 增加更多流派模板
- [ ] **用户偏好记忆**：记住用户的风格偏好、调性偏好、速度偏好
