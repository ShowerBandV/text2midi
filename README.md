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
Composer 人格层
   ├── ComposerDNA       选择作曲家人格 (8 种 Archetypes)
   └── SongDNA           跨段落动机记忆
   ↓
Go 后处理 (受 DNA 控制)
   ├── ChordPad          和弦铺底
   ├── MotifDevelopment  动机变奏 (按人格强度)
   ├── CreativeChaos     故意犯错 (按人格 Chaos 值)
   └── VoiceCrossingFix  可选 (按人格 AllowVoiceCrossing)
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
│   │   │   ├── dna.go          ComposerDNA 人格系统 + SongDNA
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
│   ├── 编曲方法/                编曲知识库
│   │   ├── liuxing.md          流行歌编曲
│   │   ├── rap.md              说唱编曲
│   │   └── rock.md             摇滚编曲
│   └── midi_output/            已生成 MIDI 文件
│
└── frontend/
    └── index.html              Web 界面
```

## ComposerDNA — 作曲家人格系统

每个 MIDI 文件都有一个 **作曲家人格**，决定它的"性格"。

```go
type ComposerDNA struct {
    MotifObsession      float64  // 多执着于重复动机
    RepetitionTolerance float64  // 允许重复的程度
    HarmonicAggression  float64  // 和声激进程度
    SilenceTolerance    float64  // 留白多少
    Chaos               float64  // 故意犯错概率
    SyncopationBias     float64  // 切分倾向
    RegisterJumpBias    float64  // 音区跳跃倾向
    AllowVoiceCrossing  bool     // 允许声部交叉
}
```

### 内置 Archetypes

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

## 生成流程 (15 阶段)

```
Phase 1: ParseIntent          LLM temp=0.2   → 风格/情绪/特征向量
Phase 2: PlanSong              LLM temp=0.7   → 和弦进行 + 结构
Phase 3: PlanArrangement       LLM temp=0.7   → 乐器分配
Phase 4: GeneratePatterns      LLM temp=0.8   → 鼓 (16步)
Phase 5: GenerateMelodyNotes   LLM temp=0.85  → 主旋律
Phase 6: GenerateBassFromMelody LLM temp=0.8  → 贝斯
Phase 7: ChordPad              规则生成        → 和弦铺底
Phase 8: MotifDevelopment      Go             → 动机变奏
Phase 9: CreativeChaos         Go             → 故意犯错
Phase 10: VoiceCrossingFix     Go (可选)       → 声部修复
Phase 11: RenderMIDI           Go             → .mid 文件
```

## 核心能力

### 旋律
- LLM 直接写, 浮点位置, 不限网格
- 动机发展 (6 种变奏)
- 音区渐高 (verse 低 → chorus 高)
- 切分节奏
- Call-Response 问答句
- 弱起准备拍

### 节奏
- Swing 精确控制: mpc58/mpc62/triplet/shuffle55
- 17 个鼓模板 (rock/metal/pop/hip-hop/funk/ambient)
- 能量曲线 → 鼓联动
- Bass-Kick 对齐

### 和声
- 和弦扩展: Cmaj7, Dm7, G7, Am9, Fsus4, Cdim, C/G
- Voice Leading (共同音保持)
- 和弦转位 (根音/一转/二转)
- 和弦代换 (副属和弦)

### 风格
- 30+ 音乐风格 + 特征向量
- 三套编曲知识库 (流行/摇滚/说唱)
- 特征向量: 7 维度

## MIDI 格式

- SMF Type 1 (多轨同步)
- Program Change / Volume (CC7) / Pan (CC10)
- Pitch Bend (14-bit)
- Expression (CC11) / Sustain (CC64) / Modulation (CC1)
- Tempo / Time Signature

## 编曲方法

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
