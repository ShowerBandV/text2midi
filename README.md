# text2midi

AI 驱动的 MIDI 作曲引擎。用自然语言描述音乐风格，自动生成完整的可编辑 MIDI 编曲。

**核心理念**：不是"生成音乐"，而是用 DNA 模板 + 风格统计特征驱动 4 个轻量级生成器，像人会记住旋律一样工作。

纯 Go 实现，零外部依赖。

---

## 快速开始

```bash
export OPENAI_API_KEY=sk-xxxxx

cd backend
go build ./cmd/generate/
go run ./cmd/generate/ \
  --prompt "周杰伦 彩虹 风格, 温暖钢琴叙事曲" \
  --style jaychou \
  --bpm 72 \
  --key "C major" \
  --bars 8 \
  --out ./midi_output
```

---

## 架构

```
用户文本 (Prompt)
   ↓
LLM Agent (2 次调用)
   ├── ParseIntent → 风格/情绪/特征向量
   └── PlanSong    → 和弦/调性/BPM
   ↓
MusicDNA Template Lookup
   ├── 从磁盘查询同风格模板
   ├── 提取和弦进行 + 动机模式
   └── 构建 StyleProfile（统计特征）
   ↓
4 × 轻量级生成器（0 次 LLM，Midra 风格 Go 端口）
   ├── GenerateLeadMidra   → scale-degree 随机动机 + 级进偏置 + 随机力度/时值
   ├── GenerateBassMidra   → 和弦根音 + 八度跳 + 随机力度/时值
   ├── GenerateChordsMidra → block/arp 模式交替 + 随机力度/时值
   └── GenerateDrumsMidra  → kick-snare-hihat + 随机变奏
   ↓
MIDI Render + MusicDNA 分析
```

**原则**：4 个生成器 < 150 行每文件，无人格系统，无后处理，无重生成循环。音乐性来自统计偏置，不是规则堆砌。

---

## 核心系统

### MusicDNA 模板库

45 首周杰伦 MIDI 被提取为结构化 DNA 模板（`templates/jaychou/`），包含：

- 和弦进行（每小节的 chord symbol）
- 动机模式（最频繁的 interval sequence）
- 段落结构（intro/verse/chorus 分界）
- 节奏密度、块 vs 分解比例

生成时按风格查询，取匹配模板的和弦进行和统计特征。

### Style Profile

从同风格的所有模板中提取统计特征，偏置生成器的随机选择：

| 特征 | 作用 |
|------|------|
| Chord Preference | 最常用前 8 和弦 |
| Interval Bias | 最常见 6 种音程 |
| BlockVsArpRatio | 块式和弦 vs 分解的比例 |
| VelocityRange | 力度范围 |
| DurationWeights | 时值分布 |

### MusicDNA Extractor

每首生成的 MIDI 自动提取：
- 和弦进行（bar-level chord detection）
- 动机模式（sliding window + frequency scoring）
- 段落结构（energy/density 变化检测）

---

## 项目结构

```
text2midi/
├── backend/
│   ├── cmd/generate/main.go      CLI 入口
│   ├── internal/
│   │   ├── agent/                LLM Agent（意图/和弦）
│   │   ├── composer/
│   │   │   ├── song.go           4 × Midra 生成器
│   │   │   └── motif_engine.go   动机引擎
│   │   ├── musicdna/
│   │   │   ├── types.go          MusicDNA 数据结构
│   │   │   ├── extractor.go      MIDI → DNA 提取器
│   │   │   ├── template.go       模板存储/查询
│   │   │   └── profile.go        Style Profile（统计特征）
│   │   ├── llm/                  Prompt 模板
│   │   ├── midi/                 SMF Type 1 写入/读取
│   │   ├── schema/               数据类型
│   │   └── style/                风格数据库
│   ├── scripts/import_dna/       MIDI → 模板导入工具
│   ├── templates/jaychou/        45 首周杰伦 DNA 模板
│   └── midi_output/              已生成 MIDI
```

---

## 导入自定义 MIDI 模板

```bash
cd backend
go run ./scripts/import_dna/ /path/to/midi/files ./templates
```

支持中文文件名，自动检测主旋律音轨。

---

## 技术栈

- **语言**: Go 1.22+
- **LLM**: OpenAI 兼容 API（默认 DeepSeek）
- **MIDI**: 原生 SMF Type 1 写入/读取器，零外部依赖

---

## License

MIT
