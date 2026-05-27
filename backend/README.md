# text2midi

AI 驱动的 MIDI 音乐生成引擎。用自然语言描述想要的音乐风格，自动生成完整的 MIDI 编曲。

纯 Go 实现，零外部依赖，编译为静态二进制文件。

---

## 快速开始

```bash
# 构建
cd backend
go build ./cmd/generate/

# 生成一首 MIDI
go run ./cmd/generate/ \
  --prompt "史诗管弦战斗音乐，英雄铜管，急速弦乐" \
  --style trap --bpm 140 --key "D minor" --bars 8 \
  --out ./midi_output

# 启动 HTTP 服务
go run ./cmd/server/

# 运行测试
go test ./...
```

### 环境变量

```bash
OPENAI_API_KEY=sk-xxxxx              # 必需
OPENAI_MODEL=deepseek-chat            # 可选，默认 deepseek-chat
OPENAI_BASE_URL=https://api.deepseek.com/v1  # 可选
```

---

## 架构

```
用户文本
   ↓
LLM Agent 流水线 (4 次调用)
   ├── ParseIntent      解析意图 + 特征向量
   ├── PlanSong         生成和弦进行 + BPM + 调性
   ├── PlanArrangement  分配乐器 (5-8 轨)
   └── GeneratePatterns 鼓节奏 (16 步网格)
   ↓
LLM 直接作曲 (2 次调用)
   ├── GenerateMelodyNotes  主旋律 (浮点位置，不限网格)
   └── GenerateBassFromMelody 贝斯跟随旋律
   ↓
Go 编曲后处理
   ├── ChordPad        和弦铺底 (规则生成)
   ├── EnergyCurve     能量曲线 → 鼓 velocity 联动
   ├── DensityControl  旋律密时和弦减薄
   ├── VoiceCrossing   bass < chords < lead 音域检测
   ├── BassKickAlign   贝斯对齐底鼓
   └── CreativeChaos   故意制造“人类错误”
   ↓
MIDI 渲染 → .mid 文件
```

## 完整生成流程

### 输入层

用户输入文本 prompt，例如：`"周杰伦风格, 中国风 R&B, 钢琴旋律, 节奏感强"`

### LLM Agent 层（4 次调用 + 2 次直接作曲）

```
Step 1: ParseIntent
  → 调用 LLM (temp=0.2)
  → 输出: style=["周杰伦","中国风","R&B"], mood=["动感","中板"],
           feature_vector={darkness=0.30, energy=0.70, density=0.60, ...}
  → 提取: 主要风格、情绪标签、7维特征向量

Step 2: PlanSong
  → 调用 LLM (temp=0.7)
  → 输入: intent JSON
  → 输出: song_plan {title, bpm, key, chord_progression, sections}
  → 和弦进行: [Dm7, G7, Cmaj7, Am7, ...]  (支持扩展和弦)
  → 替换: Chord Substitution 引擎将平庸和弦替换为副属和弦

Step 3: PlanArrangement
  → 调用 LLM (temp=0.7)
  → 输入: intent + song_plan
  → 输出: tracks = [piano, bass, chords, lead, drums, ...]
  → 每轨: name, channel, program (GM音色号), role

Step 4: GeneratePatterns (鼓)
  → 调用 LLM (temp=0.8)
  → 输入: style + bpm + key + bars
  → 输出: 16-step drum pattern {kick, snare, hihat}
  → 风格派发: 根据特征向量自动选择 rock/metal/pop/hip-hop 模板
```

### LLM 直接作曲层

```
Step 5: GenerateMelodyNotes (主旋律)
  → 调用 LLM (temp=0.85)
  → 输入: instrument="lead" + key + scale + chord_progression + feature_vector
  → 输出: 80-120 个音符, 任意浮点位置 (不限 16 步网格)
  → 每个音符: {pitch, start_beat, duration_beat, velocity, articulation}

  后处理链 (Go, 不依赖 LLM):
  5a. MotifDevelopment
      → 提取前 4 个音作为核心动机
      → 隔段变奏: 奇数段保留 LLM 原版, 偶数段做变奏
      → 变奏类型: transpose+5th, transpose-4th, invert, rhythm_double, fragment, sequence

  5b. RegisterExpansion
      → 段落 0: 原位 | 段落 1: +2 半音 | 段落 2: +5 半音 | 段落 3+: +7 半音

  5c. Syncopation
      → ~20% 的音符偏移 0.5 拍, 制造切分节奏

  5d. CallResponse
      → 偶数小节 = CALL (velocity × 1.1) | 奇数小节 = RESPONSE (velocity × 0.75)

  5e. Anacrusis
      → 如果旋律从正拍开始, 整体偏移 0.5 拍, 制造弱起感

  5f. Groove (精确 Swing)
      → mpc58 / mpc62 / triplet / shuffle55
      → 16 分音符偏移 + 人力化随机微调 + 强拍重音 + 乐句末尾 rubato

  5g. CounterMelody
      → 基于主旋律生成三度/六度副旋律
      → 取主旋律 ~40% 的音符, 在下方三度或六度复制

Step 6: GenerateBassFromMelody (贝斯跟随旋律)
  → 调用 LLM (temp=0.8)
  → 输入: 主旋律摘要 (首 5 个音 + 音域 + 高频音统计)
  → 输出: 贝斯音符, 根音锁定主旋律重拍
  → Bass-Kick Alignment: 贝斯 StartBeat 对齐最近 Kick 起音
```

### Go 编曲后处理层

```
Step 7: GenerateChordPad
  → 规则生成, 非 LLM
  → 根据和弦进行和特征向量生成和弦铺底
  → Voicing: tension 控制 7/9 音扩展, darkness 控制转位, density 控制开放排列

Step 8: VoiceCrossingFix
  → 计算 bass/chords/lead 三轨平均音高
  → 如果 bass > chords 或 chords > lead, 下移八度

Step 9: StructureTemplate
  → 选择 pop_ballad / rock_anthem / epic_cinematic 模板
  → Intro(0.3x) → Verse(0.7x) → Pre(0.9x) → Chorus(1.4x) → Bridge(0.6x) → Outro(0.3x)

Step 10: CreativeChaos
  → 根据 tension 概率决定 "人类错误" 密度
  → Blue Note: pitch -1 (降低半音)
  → Suspension: duration +0.25 beats (延音挂留)
  → Anticipation: start -0.1 beats (抢拍)

Step 11: RenderMIDI
  → 写入 Standard MIDI File Type 1
  → BPM + Time Signature + Program Change + Note On/Off + CC 事件
```

### 输出层

```
.mid 文件 → 直接拖入 FL Studio / Ableton / Logic / Cubase
Base64 → 前端实时预览
File Store → 元数据存档 + 完整性校验
```

### 15 阶段流水线

| 阶段 | 名称 | 说明 |
|------|------|------|
| 1 | ParseIntent | LLM 解析文本 -> 风格/情绪/特征向量 |
| 2 | PlanSong | 和弦进行 + 调性 + BPM + 结构 |
| 3 | PlanArrangement | 乐器分配 (5-8 轨) |
| 4 | GeneratePatterns | 鼓 (16 步网格) |
| 5 | GenerateMelodyNotes | LLM 直接写主旋律 |
| 6 | MotifDevelopment | 动机提取 + 隔段变奏 |
| 7 | RegisterExpansion | 段落音区渐高 |
| 8 | Syncopation | 切分节奏 |
| 9 | CallResponse | 问答句力度 |
| 10 | Anacrusis | 弱起准备拍 |
| 11 | Groove | 精确 Swing (MPC 58%) |
| 12 | CounterMelody | 副旋律 |
| 13 | GenerateBassFromMelody | 贝斯跟随旋律 |
| 14 | ChordPad + Post-processing | 和弦 + 能量曲线 + 声部修复 + 随机混沌 |
| 15 | RenderMIDI | 写入 .mid 文件 |

---

## 项目结构

```
backend/
├── cmd/
│   ├── generate/main.go    CLI 生成器
│   └── server/main.go      HTTP API 服务器
│
├── internal/
│   ├── agent/           LLM Agent 链
│   ├── composer/        编曲后处理引擎
│   │   ├── energy.go    能量曲线 + Bass-Kick 对齐 + 声部检测
│   │   ├── groove.go    Swing 量化 + 乐句呼吸
│   │   ├── motif.go     动机发展 + 副旋律 + 问答句 + 弱起 + 音区展开 + 切分
│   │   ├── structure.go 段落模板 + 动态范围
│   │   └── substitution.go 和弦代换
│   │
│   ├── generator/       规则乐器生成器
│   │   ├── drum.go      鼓 (kick/snare/hihat + crash/tom fill)
│   │   ├── bass.go      Walking bass / 长音保持
│   │   ├── chord.go     扩展和弦 + Voice Leading
│   │   ├── lead.go      乐句弧线旋律
│   │   ├── rhythm_guitar.go Power chord + 主音吉他
│   │   └── humanize.go  乐器差异化 timing/velocity
│   │
│   ├── harmony/         和声约束 + Voice Leading
│   ├── llm/             Prompt 模板 + LLM 客户端
│   ├── midi/            原生 SMF Type 1 写入
│   ├── music/           乐理工具 (音阶/和弦/音高)
│   ├── mutation/        Mutation 引擎 + Creative Chaos
│   ├── schema/          数据类型
│   ├── store/           文件存储
│   └── style/           风格数据库 + 配器模板
│
├── frontend/index.html  Web 界面
└── midi_output/         已生成 MIDI
```

---

## 核心能力

### 旋律
- LLM 直接写旋律，不受 16 步网格限制
- 动机发展 (6 种变奏: 移调/倒影/模进/碎片化/节奏加倍/序列)
- 音区渐高 (verse 低 → chorus 高)
- 切分节奏 (~20% 音符 offbeat)
- Call-Response 问答句
- 弱起准备拍
- 三度/六度副旋律
- Articulation 演奏法落地 (staccato/legato/accent/bend/pizzicato/tremolo)

### 节奏
- Swing 精确控制: mpc58 / mpc62 / triplet / shuffle55
- 17 个鼓模板 (rock/metal/pop/hip-hop/funk/ambient/cinematic/latin)
- 能量曲线 → 鼓 velocity + crash 联动
- Bass-Kick 对齐

### 和声
- 和弦扩展: Cmaj7, Dm7, G7, Am9, Fsus4, Cdim, Caug, Cm7b5, C/G
- Voice Leading (共同音保持 + 最小移动)
- 和弦转位 (根音 → 一转 → 二转)
- 平行五度/八度检测
- 和弦代换 (副属和弦)

### 表达
- 乐器差异化 humanize (钢琴/吉他/铜管/弦乐/鼓)
- Creative Chaos (蓝色音符 / 延音挂留 / 抢拍)
- CC11 表情曲线
- 宽动态范围 (intro=0.3x, climax=1.5x)
- 段落模板 (pop_ballad / rock_anthem / epic_cinematic)

### 风格
- 30+ 音乐风格 + 特征向量
- 配器模板 (rock/pop/hip-hop/cinematic/chinese/electronic/jazz)
- 特征向量: 7 维度 (暗度/能量/原声度/密度/节奏复杂度/张力/低保真度)

---

## MIDI 格式支持

- SMF Type 1 (多轨同步)
- Program Change / Volume (CC7) / Pan (CC10)
- Pitch Bend (14-bit)
- Expression (CC11) / Sustain (CC64) / Modulation (CC1)
- Tempo / Time Signature 事件

---

## 开发

```bash
# 全部测试
go test ./...

# 构建
go build ./...

# 生成 (简写参数)
go run ./cmd/generate/ --prompt "一首歌" --out ./midi_output
```

生成的 `.mid` 文件直接拖入 FL Studio / Ableton / Logic / Cubase 即可使用。
