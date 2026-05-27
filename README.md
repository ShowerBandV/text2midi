# text2midi 🎵

AI 驱动的 MIDI 音乐生成引擎。用自然语言描述想要的音乐风格，自动生成完整的 MIDI 编曲。

纯 Go 实现，零外部 MIDI 依赖。提供 CLI 生成器 + HTTP API 服务 + Web 前端。

---

## 快速开始

### 1. 设置 API Key

```bash
export OPENAI_API_KEY=sk-xxxxx
# 可选：切换模型 / 服务商
export OPENAI_MODEL=deepseek-chat
export OPENAI_BASE_URL=https://api.deepseek.com/v1
```

### 2. CLI 生成 MIDI

```bash
cd backend
go run ./cmd/generate/ \
  --prompt "epic orchestral battle theme, heroic brass, rapid strings" \
  --style trap --bpm 140 --key "D minor" --bars 8 \
  --out ./midi_output
```

### 3. 启动 Web 服务

```bash
cd backend
go run ./cmd/server/
# → http://localhost:8080
# 打开 frontend/index.html 即可使用
```

### 4. 运行测试

```bash
cd backend
go test ./...
```

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/
│   │   ├── generate/main.go     CLI 生成器
│   │   └── server/main.go       HTTP API 服务
│   ├── internal/
│   │   ├── agent/               LLM Agent 链 (意图解析/和弦/配器/模式生成)
│   │   ├── composer/            编曲后处理引擎 (groove/动机/结构/能量/和弦代换)
│   │   ├── generator/           规则乐器生成器 (bass/chord/drum/lead/rhythm_guitar)
│   │   ├── harmony/             和声约束 + Voice Leading
│   │   ├── llm/                 Prompt 模板 + LLM 客户端
│   │   ├── midi/                原生 SMF Type 1 写入 (零依赖)
│   │   ├── music/               乐理工具 (音阶/和弦/音高)
│   │   ├── mutation/            Creative Chaos 引擎
│   │   ├── schema/              核心数据类型
│   │   ├── store/               文件存储 + 元数据管理
│   │   └── style/               风格数据库 (40+ 风格)
│   ├── go.mod
│   └── generated/               API 生成的 MIDI 输出
│
├── frontend/
│   └── index.html               Web 界面 (单页 HTML/JS)
│
└── README.md
```

---

## API 服务

启动后端后暴露三个接口：

| 接口 | 方法 | 说明 |
|---|---|---|
| `/api/info` | GET | 返回可用风格列表 + 各 tier 限制 |
| `/api/generate` | POST | 生成 MIDI，返回文件元数据 |
| `/api/files/{id}` | GET | 下载生成的 `.mid` 文件 |

### POST /api/generate

```json
{
  "prompt": "dark trap beat, heavy 808 sliding bass",
  "style": "trap",
  "bpm": 140,
  "key": "D minor",
  "bars": 8,
  "tier": "free",
  "seed": 42
}
```

---

## 技术栈

- **后端**: Go 1.22+
- **LLM**: OpenAI 兼容 API (默认 DeepSeek Chat)
- **MIDI**: 原生 SMF Type 1 写入，零外部依赖
- **前端**: 纯 HTML/CSS/JS，无框架

---

## License

MIT
