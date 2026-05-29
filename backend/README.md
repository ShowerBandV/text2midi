# text2midi Backend

Go 语言 MIDI 音乐生成引擎。支持 CLI 和 HTTP API。

**两种模式**：
- **LLM 模式** — DeepSeek/GPT 驱动，解析 prompt → 规划 → 编曲 → 评审迭代
- **Local 模式** — 纯 Go 规则引擎，零 API 调用，秒出 .mid

## 快速开始

```bash
cd backend

# 本地模式（无需 API Key，纯规则引擎）
go run ./cmd/generate/ --local
go run ./cmd/generate/ --local --style "punk" --key "E minor" --bpm 180
go run ./cmd/generate/ --local --style "emo" --key "A minor" --bpm 72 --bars 32

# LLM 模式（需 API Key）
export OPENAI_API_KEY=sk-xxx
go run ./cmd/generate/ --prompt "cyberpunk synth lead in C minor"

# 启动 HTTP 服务
go run ./cmd/server/
```

### CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--prompt` | — | 音乐描述（LLM 模式必填） |
| `--style` | `trap` | 风格（local 模式支持：rpg / emo / rock / punk / metal / pop / trap / ambient） |
| `--key` | `C minor` | 调性，如 `C major`、`A minor`、`E minor` |
| `--bpm` | `140` | 速度 |
| `--bars` | `8` | 小节数 |
| `--out` | `./midi_output` | 输出目录 |
| `--local` | `false` | 启用本地规则引擎模式（无需 API Key） |

### Local 模式风格与编制

不同风格自动匹配不同的乐器编制和参数：

| 风格 | 编制 | dark | energy | BPM |
|------|------|------|--------|-----|
| **rpg** | Drums + Bass + Pad + Piano + Strings | 0.20 | 0.45 | 100 |
| **emo** | Drums + Bass + Pad + Piano + Strings | 0.75 | 0.32 | 72 |
| **rock** | Drums + Bass + Rhythm Gtr + Lead Gtr | 0.45 | 0.70 | 130 |
| **punk** | Drums + Bass + Rhythm Gtr + Lead Gtr | 0.35 | 0.78 | 170 |
| **metal** | Drums + Bass + Rhythm Gtr + Lead Gtr | 0.80 | 0.85 | 160 |
| **pop** | Drums + Bass + Pad + Piano + Strings | 0.25 | 0.60 | 120 |
| **trap** | Drums + 808 Bass + Pad + Synth Lead | 0.55 | 0.55 | 140 |
| **ambient** | Drums + Bass + Sweep Pad + Lead Pad | 0.30 | 0.18 | 60 |

## Token 消耗与成本（LLM 模式）

LLM 模式下每次 API 调用自动追踪 token 消耗，生成结束后显示汇总：

```
  ═══ Token Usage ═══
  Calls:       6
  Input:       12,345 tokens
  Output:      4,567 tokens
  Total:       16,912 tokens
  Model:       deepseek-chat
  Cost:        $0.0083 USD  (≈ ¥0.0602 CNY)
```

内置定价按模型自动匹配：

| 模型 | 输入/1M tokens | 输出/1M tokens |
|------|---------------|---------------|
| deepseek-chat | $0.27 | $1.10 |
| deepseek-reasoner | $0.55 | $2.19 |
| gpt-4 | $30.00 | $60.00 |
| gpt-3.5 | $0.50 | $1.50 |
| claude | $15.00 | $75.00 |

可通过环境变量覆盖：`LLM_INPUT_COST_PER_1M=0.14 LLM_OUTPUT_COST_PER_1M=0.56`

## API

| 端点 | 说明 |
|---|---|
| `GET /api/info` | 风格列表 + tier 限制 |
| `POST /api/generate` | 生成 MIDI |
| `GET /api/files/{id}` | 下载 .mid 文件 |
| `POST /api/dna/extract` | 从 events 提取 MusicDNA |
| `GET /api/dna/library` | 列出 DNA 模板 |
| `GET /api/dna/library/{name}` | 获取 DNA 模板详情 |

## 项目结构

```
backend/
├── cmd/
│   ├── generate/       CLI 生成器
│   ├── server/         HTTP API 服务
│   └── dnascan/        DNA 扫描工具
├── internal/
│   ├── agent/          LLM Agent 链
│   ├── composer/       编曲引擎 + DNA 映射
│   ├── generator/      规则生成器 (bass/chord/drum/lead/fx)
│   ├── llm/            LLM 客户端
│   ├── midi/           原生 MIDI 写入
│   ├── musicdna/       DNA 提取/评分/序列化
│   ├── schema/         数据类型
│   ├── store/          文件存储
│   └── style/          40+ 风格数据库
└── .env.example        环境变量模板
```

## 环境变量

```bash
# LLM 模式
OPENAI_API_KEY=sk-xxx          # API Key（必填）
OPENAI_MODEL=deepseek-chat     # 模型名
OPENAI_BASE_URL=...            # API 地址（默认 DeepSeek）

# Token 成本（可选，覆盖内置定价）
LLM_INPUT_COST_PER_1M=0.27     # 输入每百万 token 价格（USD）
LLM_OUTPUT_COST_PER_1M=1.10    # 输出每百万 token 价格（USD）

# 评审迭代（可选）
LLM_MAX_REFINE_ROUNDS=2        # reviewer/leader 最大迭代轮数（默认 2）

# 服务
PORT=8080                      # HTTP 服务端口
```
