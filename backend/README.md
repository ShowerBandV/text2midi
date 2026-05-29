# text2midi — AI 游戏音乐生成引擎

Go 语言 MIDI 音乐生成引擎。**LLM 决策 + Go 规则引擎展开**，支持 CLI 和 HTTP API。

两种模式：
- **Local 模式** — 纯 Go 规则引擎，8 种风格专属生成器，零 API 调用，秒出 .mid
- **LLM 模式** — DeepSeek/GPT 驱动，解析意图 → 规划 → 编曲 → 并行 Agent 生成

默认歌曲结构：Intro → Verse → Chorus → Bridge → Chorus → Outro，约 2 分钟。

## 快速开始

```bash
# 本地模式（无需 API Key）
go run ./cmd/generate/ --local
go run ./cmd/generate/ --local --style "metal" --key "E minor" --bpm 160
go run ./cmd/generate/ --local --style "pop" --key "C major" --bpm 110 --pentatonic
go run ./cmd/generate/ --local --style "trap" --key "C minor" --bpm 140

# 先看 plan 再生成
go run ./cmd/generate/ --local --style "metal" --dry-run

# 统一力度
go run ./cmd/generate/ --local --style "punk" --flat-vel 100

# LLM 模式（需 API Key）
export OPENAI_API_KEY=sk-xxx
go run ./cmd/generate/ --prompt "cyberpunk synth lead in C minor"

# LLM 评审迭代
go run ./cmd/generate/ --prompt "epic boss battle" --refine

# HTTP 服务
go run ./cmd/server/
```

## CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--prompt` | — | 音乐描述（LLM 模式必填） |
| `--style` | `trap` | 风格：rpg / emo / rock / punk / metal / pop / trap / ambient |
| `--key` | `C minor` | 调性，如 `C major`、`A minor` |
| `--bpm` | 风格默认 | 速度（各风格有合理默认值） |
| `--bars` | 风格默认 | 小节数（默认约 2 分钟长度） |
| `--out` | `./midi_output` | 输出目录 |
| `--local` | `false` | 本地规则引擎模式 |
| `--dry-run` | `false` | 打印 plan 后退出 |
| `--refine` | `false` | LLM 评审迭代（LLM 模式） |
| `--resume` | `false` | 从 checkpoint 恢复（LLM 模式） |
| `--pentatonic` | `false` | 五声音阶 + 中国风装饰 |
| `--flat-vel` | `0` | 统一所有音符力度到此值（0=禁用） |

## 风格一览

| 风格 | 编制 | BPM | Bars | 时长 | 主音特点 |
|------|------|-----|------|------|---------|
| **metal** | Drums + Bass + Rhythm Gtr + Lead Gtr + Harmony Gtr | 160 | 72 | ~1:48 | Bodom 风格：riff→arpeggio→solo→silence |
| **punk** | Drums + Bass + Rhythm Gtr + Lead Gtr | 170 | 24 | ~0:34 | D-beat + 直八分音符全下拨 |
| **rock** | Drums + Bass + Rhythm Gtr + Lead Gtr | 130 | 24 | ~0:44 | Backbeat + off-beat push |
| **pop** | Drums + Bass + Pad + Piano + Strings | 110 | 64 | ~2:19 | John Legend 左手八度+右手和弦+旋律 |
| **rpg** | Drums + Bass + Pad + Piano + Strings | 100 | 48 | ~1:55 | 同上 |
| **emo** | Drums + Bass + Pad + Piano + Strings | 72 | 24 | ~1:20 | Sparse rim-click + 长音 sustain |
| **trap** | Drums + 808 Bass + Pad + Synth Lead | 140 | 16 | ~0:27 | 808 sub-bass + 16th hi-hat roll |
| **ambient** | Drums + Bass + Sweep Pad + Lead Pad | 60 | 16 | ~1:04 | 开放排列 wide interval |

## 各风格鼓对比

| 风格 | Verse 镲 | Chorus 镲 | Kick | Snare | Fill |
|------|---------|----------|------|-------|------|
| **punk** | 闭镲 driving | 开镲全 wash + crash | D-beat 6 连 | Ghost drag + flam | Tom / KS 交替 / Flam |
| **metal** | Ride bell | China + 开镲 | 双踩 16th + 反拍 | Ghost drag + flam | 五连 tom 下行 |
| **rock** | Ride | Crash 正拍 + 开镲 | 1/3 + 反拍 push | Ghost drag + flam | Tom / KS 交替 |
| **pop** | 闭镲 swing | 开镲 wash | 1/3 + push | Rim-click(verse) / Full(chorus) | Snare roll 加速 |
| **trap** | — | — | Sparse 硬击 | Clap 2/4 | 32nd hi-hat roll |

## Token 消耗与成本（LLM 模式）

生成结束后自动显示：

```
  ═══ Token Usage ═══
  Calls:       6
  Input:       12,345 tokens
  Output:      4,567 tokens
  Total:       16,912 tokens
  Model:       deepseek-chat
  Cost:        $0.0083 USD  (≈ ¥0.0602 CNY)
```

| 模型 | 输入/1M | 输出/1M |
|------|---------|---------|
| deepseek-chat | $0.27 | $1.10 |
| deepseek-reasoner | $0.55 | $2.19 |
| gpt-4 | $30.00 | $60.00 |
| gpt-3.5 | $0.50 | $1.50 |
| claude | $15.00 | $75.00 |

覆盖：`LLM_INPUT_COST_PER_1M=0.14 LLM_OUTPUT_COST_PER_1M=0.56`

## API

| 端点 | 说明 |
|---|---|
| `GET /api/info` | 风格列表 + tier 限制 |
| `POST /api/generate` | 生成 MIDI |
| `GET /api/files/{id}` | 下载 .mid |
| `POST /api/dna/extract` | 从 events 提取 MusicDNA |
| `GET /api/dna/library` | DNA 模板列表 |
| `GET /api/dna/library/{name}` | DNA 模板详情 |

## 项目结构

```
├── cmd/generate/         CLI 生成器
├── cmd/server/           HTTP API 服务
├── cmd/dnascan/          DNA 扫描工具
├── internal/
│   ├── agent/            LLM Agent 链（ParseIntent / PlanSong / PlanArrangement / Reviewer / Leader）
│   ├── composer/         编曲引擎（songSection / MelodyGrammar / DNA / motif / phrase）
│   ├── generator/        规则生成器（bass / chord / drum / lead / fx）
│   ├── llm/              LLM 客户端 + prompt 模板 + token 追踪
│   ├── midi/             原生 MIDI 写入
│   ├── musicdna/         DNA 提取 / 评分 / 序列化
│   ├── schema/           数据类型
│   ├── store/            文件存储
│   └── style/            风格数据库
├── knowledges/
│   └── chords.md         和弦进行知识库（可独立编辑）
├── templates/jaychou/    周杰伦风格模板
├── ROADMAP.md            待改进路线图
└── architecture.md       架构设计文档
```

## 环境变量

```bash
# LLM
OPENAI_API_KEY=sk-xxx          # API Key
OPENAI_MODEL=deepseek-chat     # 模型名
OPENAI_BASE_URL=...            # API 地址（默认 DeepSeek）

# Token 成本（覆盖内置定价）
LLM_INPUT_COST_PER_1M=0.27
LLM_OUTPUT_COST_PER_1M=1.10

# 迭代
LLM_MAX_REFINE_ROUNDS=2

# 服务
PORT=8080
```

## 对比 Midra / Clef

| 维度 | text2midi | Midra | Clef |
|------|:---:|:---:|:---:|
| 风格专属生成器 | ✅ 8 种 | ❌ | ❌ |
| 歌曲结构 | ✅ 自动分段 | ❌ | ✅ 手动 |
| 离线生成 | ✅ `--local` | ✅ `--note-mode rule` | ❌ |
| 并行生成 | ✅ goroutine | ✅ ThreadPool | ✅ Agent Teams |
| Token 追踪 | ✅ | ❌ | ✅（平台自带） |
| 音频渲染 | ❌ | ✅ MIDI→WAV→MP3 | ✅ SF2 播放 |
| 验证体系 | 基础 | ✅ | ✅ music21 |
| SF2 感知 | ❌ | ❌ | ✅ |
| ABC 记谱 | ❌ | ❌ | ✅ |

详见 [ROADMAP.md](ROADMAP.md)。
