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
  --prompt "周杰伦风格, 中国风 R&B, 钢琴旋律, 节奏感强" \
  --style lofi \
  --bpm 85 \
  --key "C major" \
  --bars 8 \
  --out ./midi_output

# 运行测试
go test ./...
```

## 架构

```
用户文本
   ↓
LLM Agent 链 (6 次调用)
   ├── ParseIntent       解析意图 + 特征向量
   ├── PlanSong          生成和弦进行 + 调性 + BPM
   ├── PlanArrangement   分配乐器 (4-8 轨)
   ├── GeneratePatterns  鼓节奏 (风格自动派发)
   ├── GenerateMelody    主旋律 (LLM 直接写, 不限网格)
   └── GenerateBass      贝斯跟随旋律
   ↓
Composer 引擎
   ├── ComposerDNA       选择作曲家人格 (8 种 Archetypes)
   ├── SongMemory        全曲动机记忆 + 主题回归
   ├── MotifDevelopment  动机变奏 (6 种变奏类型)
   ├── Transition Engine 段落过渡 (drum_fill/riser/silence/breath)
   ├── Dynamic Layering  能量控制乐器数量
   ├── Texture Layer     氛围铺底 (drone/pad/noise/arp)
   ├── CreativeChaos     故意犯错
   └── Stem Exporter     分轨导出 (游戏引擎集成)
   ↓
MIDI 渲染 → .mid 文件
```

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/
│   │   ├── generate/main.go    CLI 生成器
│   │   └── server/main.go      HTTP API
│   ├── internal/
│   │   ├── agent/              LLM Agent 链
│   │   ├── composer/           编曲引擎
│   │   │   ├── dna.go          ComposerDNA 人格 + SongMemory 全曲记忆
│   │   │   ├── transition.go   段落过渡 (6 种类型)
│   │   │   ├── dynamics.go     动态层叠 (乐器数量控制)
│   │   │   ├── texture.go      氛围铺底 (pad/drone/arp/noise)
│   │   │   ├── stems.go        Stem 分轨导出
│   │   │   ├── energy.go       能量曲线 + Bass-Kick 对齐
│   │   │   ├── groove.go       Swing 量化 + 乐句呼吸
│   │   │   ├── motif.go        动机发展 + 副旋律 + 切分
│   │   │   ├── structure.go    段落模板 + 动态范围
│   │   │   └── substitution.go 和弦代换
│   │   ├── generator/          规则乐器生成器
│   │   │   ├── drum.go         鼓 (kick/snare/hihat + fill)
│   │   │   ├── bass.go         Walking bass
│   │   │   ├── chord.go        扩展和弦 + Voice Leading
│   │   │   ├── lead.go         乐句弧线旋律
│   │   │   └── rhythm_guitar.go Power chord chugging
│   │   ├── harmony/            和声约束 + Voice Leading
│   │   ├── llm/                Prompt 模板 + LLM 客户端
│   │   ├── midi/               原生 SMF Type 1 写入
│   │   ├── music/              乐理工具
│   │   ├── mutation/           Mutation + Creative Chaos
│   │   ├── schema/             数据类型
│   │   ├── store/              文件存储
│   │   └── style/              风格数据库 + 配器模板
│   ├── 编曲方法/                编曲知识库 (流行/摇滚/说唱)
│   └── midi_output/            已生成 MIDI 文件
│
└── frontend/
    └── index.html              Web 界面
```

## 核心架构

### ComposerDNA — 作曲家人格

每个 MIDI 文件都有一个作曲家人格，决定它的"性格"。

| 人格 | MotifObsession | Chaos | VoiceCrossing | 适用场景 |
|------|----------------|-------|---------------|---------|
| **Toby Fox (Undertale)** | 0.90 | 0.30 | 允许 | 游戏音乐、重复动机 |
| **Nobuo Uematsu (FF)** | 0.70 | 0.10 | 允许 | JRPG、幻想 |
| **Hans Zimmer** | 0.85 | 0.20 | 不允许 | 史诗、金属、电影 |
| **JRPG Default** | 0.75 | 0.15 | 不允许 | RPG 通用 |
| **Retro Game** | 0.85 | 0.10 | 不允许 | 8bit、复古 |
| **Hyperpop Maniac** | 0.40 | 0.90 | 允许 | 实验、高能量 |
| **Classical Purist** | 0.60 | 0.05 | 不允许 | 古典、管弦 |
| **Default** | 0.50 | 0.10 | 不允许 | 通用 |

### SongMemory — 全曲作曲逻辑

记录整个曲子的动机、节奏标识、和声倾向。后续段落可以引用前面的材料。

```
Verse (A):     Motif A   → 原始动机
Chorus (A'):   Motif A   → 八度上移 + 节奏加密
Bridge (B):    Motif A   → 倒影 (上下颠倒)
Final Chorus:  Motif A   → 原始 + 副旋律
Outro:         Motif A   → 慢速、弱化
```

### Transition Engine — 段落过渡

自动在段落之间插入过渡，基于能量差选择类型：

| 过渡类型 | 触发条件 | 效果 |
|---------|---------|------|
| **drum_fill** | 同能量级过渡 | 急速军鼓 + Tom + Crash |
| **riser** | 能量大幅上升 | 上行音阶 + Crash |
| **cymbal_rev** | 能量下降 | 逐渐增强的镲片 |
| **bass_slide** | 能量下降 | 贝斯滑音 |
| **silence** | 能量骤降 | 突然静音 |
| **breath_pause** | 能量骤降 | 半拍呼吸停顿 |

### Dynamic Layering — 真·动态系统

按能量控制乐器数量，不只 velocity：

| 能量 | 活跃乐器 |
|------|---------|
| 0.0-0.3 | pad / atmosphere 仅 |
| 0.3-0.5 | + bass + hihat |
| 0.5-0.7 | + drums + chords |
| 0.7-1.0 | + lead + counter melody |

### Texture Layer — 氛围铺底

游戏音乐专用的氛围音层 (pad/drone/noise/arp/reverse)，随能量自动切换。

## 生成流程 (15 阶段)

```
Phase 1:  ParseIntent          LLM temp=0.2    → 风格/情绪/特征向量
Phase 2:  PlanSong             LLM temp=0.7    → 和弦进行 + 结构
Phase 3:  PlanArrangement      LLM temp=0.7    → 乐器分配
Phase 4:  GeneratePatterns     LLM temp=0.8    → 鼓 (16步, 风格派发)
Phase 5:  GenerateMelodyNotes  LLM temp=0.85   → 主旋律
Phase 6:  GenerateBassFromMelody LLM temp=0.8  → 贝斯跟随旋律
Phase 7:  ChordPad             规则生成         → 和弦铺底
Phase 8:  SongMemory           自动            → 记录动机 + 主题回归
Phase 9:  Transition Engine    自动            → 段落过渡
Phase 10: Dynamic Layering     自动            → 乐器数量控制
Phase 11: Texture Layer        自动            → 氛围铺底
Phase 12: MotifDevelopment     Go              → 动机变奏
Phase 13: CreativeChaos        Go              → 故意犯错
Phase 14: VoiceCrossingFix     Go (可选)       → 声部修复
Phase 15: RenderMIDI           Go              → .mid 文件
```

## 核心能力

### 旋律
- LLM 直接写, 浮点位置, 不限网格
- 动机发展 (6 种变奏: 移调/倒影/模进/碎片化/节奏加倍/序列)
- 音区渐高 (verse 低 → chorus 高)
- 切分节奏
- Call-Response 问答句
- 弱起准备拍
- 主题回归系统

### 节奏
- Swing 精确控制: mpc58/mpc62/triplet/shuffle55
- 17 个鼓模板 (rock/metal/pop/hip-hop/funk/ambient/cinematic/latin)
- 能量曲线 → 鼓联动
- Bass-Kick 对齐
- 段落过渡自动生成

### 和声
- 和弦扩展: Cmaj7, Dm7, G7, Am9, Fsus4, Cdim, C/G
- Voice Leading (共同音保持)
- 和弦转位 (根音/一转/二转)
- 和弦代换 (副属和弦)

### 动态
- 乐器数量随能量变化 (pad only → 全乐器)
- 氛围铺底自动切换
- 宽动态范围 (intro=0.3x, climax=1.5x)

### 风格
- 30+ 音乐风格 + 特征向量
- 三套编曲知识库 (流行/摇滚/说唱)
- 特征向量: 7 维度 (暗度/能量/原声度/密度/节奏复杂度/张力/低保真度)
- 8 种作曲家人格

## MIDI 格式

- SMF Type 1 (多轨同步)
- Program Change / Volume (CC7) / Pan (CC10)
- Pitch Bend (14-bit)
- Expression (CC11) / Sustain (CC64) / Modulation (CC1)
- Tempo / Time Signature

## 编曲知识库

项目内置三份专业编曲知识库 (`backend/编曲方法/`):

- **liuxing.md** — 流行歌编曲: 段落结构、动态对比、渐进加入乐器
- **rap.md** — 说唱编曲: Boom Bap / Trap / Drill 三风格
- **rock.md** — 摇滚编曲: 双吉他分工、Riff 优先、Solo 设计

这些知识被注入到 LLM Prompt 中指导生成。

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek)
- **MIDI**: 原生写入, 零外部依赖
- **前端**: 纯 HTML/JS (无框架)

## License

MIT
