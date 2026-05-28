# text2midi Frontend

React + TypeScript + Vite 前端，MIDI 音乐生成的图形界面。

## 启动

```bash
cd frontend
npm install
npm run dev
```

开发服务器默认运行在 `http://localhost:5173`，API 请求自动代理到 `http://localhost:8080`。

## 构建

```bash
npm run build     # 输出到 dist/
npm run preview   # 预览构建产物
```

## 功能

- **Generate** — 文本描述 → AI 生成 MIDI（调用 Go 后端 `/api/generate`）
- **Editor** — 钢琴卷帘编辑器，实时试听
- **Library** — 曲库管理，下载 .mid 文件

## 技术栈

- React 19 + TypeScript
- Vite 6 + Tailwind CSS 4
- Web Audio API（钢琴/合成器/弦乐仿真）
- lucide-react 图标
