import { en } from "./en";

export const zh: typeof en = {
  // Navbar
  "nav.generate": "创作",
  "nav.library": "曲库",
  "nav.signIn": "登录",
  "nav.logout": "退出登录",
  "nav.user": "用户",

  // Auth modal
  "auth.welcomeBack": "欢迎回来",
  "auth.join": "加入 MidiMind",
  "auth.signInTo": "登录你的账号",
  "auth.createFree": "创建免费账号",
  "auth.signIn": "登录",
  "auth.register": "注册",
  "auth.username": "用户名",
  "auth.enterUsername": "请输入用户名",
  "auth.password": "密码",
  "auth.enterPassword": "请输入密码",
  "auth.atLeast6": "至少 6 位字符",
  "auth.pleaseWait": "请稍候...",
  "auth.createAccount": "创建账号",
  "auth.fillBoth": "请填写用户名和密码",

  // Generate page
  "gen.title": "把文字变成",
  "gen.titleHighlight": "音乐",
  "gen.subtitle": "描述你脑海中的声音——一段忧郁的合成器琶音、一首爵士钢琴叙事曲、一段电影感的管弦气势——我们的引擎在几秒内为你生成一首可直接使用的 MIDI 作品。",
  "gen.placeholder": "一首 C 小调的忧郁爵士钢琴独奏...",
  "gen.presetCyberpunk": "赛博朋克",
  "gen.presetJazz": "爵士和弦",
  "gen.presetString": "弦乐铺底",
  "gen.generate": "生成 MIDI",
  "gen.loginRequired": "请先登录后再生成 MIDI",

  // Feature cards
  "feat.engine": "智能作曲引擎",
  "feat.engineDesc": "解析自然语言指令，分析音乐意图，生成完整的多轨编曲——鼓、贝斯、和弦、主旋律与铺底——每一步都基于乐理规则。",
  "feat.llm": "LLM 增强",
  "feat.llmDesc": "接入你自己的 LLM API Key，解锁更深的音乐智能——更丰富的和声走向、更具表现力的编曲、风格自适应的作曲能力。",
  "feat.arrangement": "完整编曲",
  "feat.arrangementDesc": "每次生成都是一份完整的编曲——鼓、贝斯、和弦、主旋律与铺底——分别放置在不同 MIDI 轨道上，可直接导入任何 DAW。",
  "feat.realtime": "实时生成",
  "feat.realtimeDesc": "高性能 MIDI 生成，即时下载。内置用户系统、生成历史与 MusicDNA 分析——让你追踪、回顾并学习每一次创作。",

  // Loading
  "load.analyzing": "正在分析提示语义...",
  "load.querying": "正在查询 MidiMind 音乐大脑...",
  "load.chords": "正在选择和弦区间...",
  "load.beats": "正在生成节拍触发...",
  "load.velocity": "正在打磨力度表现...",
  "load.synthesizing": "正在合成低延迟 MIDI 文件...",

  // Library
  "lib.title": "曲库",
  "lib.empty": "还没有作品",
  "lib.emptyHint": "前往创作页面生成你的第一首 MIDI。",
  "lib.export": "导出",
  "lib.download": "下载 .mid",
  "lib.downloadLocal": "本地下载",
  "lib.downloadServer": "从服务器下载",
  "lib.fromServer": "从服务器",
  "lib.fromServerDesc": "原始后端文件",
  "lib.share": "分享",
  "lib.copyLink": "复制链接",
  "lib.copied": "已复制！",
  "lib.preview": "预览",
  "lib.previewDesc": "浏览器音频渲染",

  // Metadata
  "meta.duration": "时长",
  "meta.key": "调性",
  "meta.tempo": "速度",
  "meta.complexity": "复杂度",
  "meta.seed": "种子",
  "meta.bpm": "BPM",

  // Library features
  "feat.chordDetection": "和弦检测",
  "feat.chordDetectionDesc": "从 MIDI 音符中检测和弦",
  "feat.regenerate": "AI 重新生成",
  "feat.regenerateDesc": "基于当前轨道创建变体",
  "feat.regenerating": "生成中...",
  "feat.regenerateBtn": "重新生成",
  "feat.dna": "DNA 分析",
  "feat.dnaDesc": "MusicDNA 结构与质量评分",
  "feat.quality": "质量评分：",

  // Info bar
  "info.styles": "风格",
  "info.tiers": "套餐",
  "info.bars": "小节",

  // Footer
  "footer.tagline": "AI 驱动 MIDI 生成 — 把你的想法变成音乐。",
  "footer.terms": "服务条款",
  "footer.privacy": "隐私政策",
  "footer.api": "API 文档",
  "footer.discord": "Discord",
  "footer.twitter": "Twitter / X",

  // Misc
  "buyMeCoffee": "请我喝杯咖啡",
};
