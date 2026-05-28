# text2midi Backend

Go 语言 MIDI 音乐生成引擎。支持 CLI 和 HTTP API。

## 快速开始

```bash
cd backend

# 本地模式（无需 API Key）
go run ./cmd/generate/ --local

# 正常模式（需 API Key）
export OPENAI_API_KEY=sk-xxx
go run ./cmd/generate/ --prompt "cyberpunk synth lead in C minor"

# 启动 HTTP 服务
go run ./cmd/server/
```

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
OPENAI_API_KEY=sk-xxx     # API Key（本地模式不需要）
OPENAI_MODEL=deepseek-chat # 模型名
OPENAI_BASE_URL=...        # API 地址
PORT=8080                  # 服务端口
```
