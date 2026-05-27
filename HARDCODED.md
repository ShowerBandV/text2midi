# Hardcoded Values Audit

所有需要后期消除的硬编码，按模块分类。

## 1. 风格判定阈值 (style detection thresholds)

这些 `if x > N` 决定音乐走哪个分支，但阈值是拍脑袋写的：

| 文件 | 行 | 问题 |
|------|-----|------|
| `song.go` | 78-85 | `darkness>0.7 && energy>0.7` 等风格切换阈值 |
| `emotion.go` | 81-94 | tension/energy/stability/warmth/brightness 判定阈值 |
| `dynamics.go` | 19-32 | DynamicLayers 能量层级阈值 [0.0, 0.3, 0.5, 0.7] |
| `texture.go` | 91-96 | SelectTexture 能量阈值 [0.2, 0.5, 0.7] |
| `transition.go` | 58-77 | 过渡选择阈值 `>0.8 && diff>0.3` 等 |
| `energy.go` | 101-106 | 和弦风格阈值 [0.7, 0.4] |
| `structure.go` | 95-106 | dynamicMap 段落力度系数 |

## 2. 鼓相关硬编码 (drum hardcodes)

鼓最容易被听出"同一个引擎"：

| 文件 | 行 | 问题 |
|------|-----|------|
| `song.go` | 99 | 鼓 velocity 公式 `60 + energy*50 + tension*20` |
| `song.go` | 101 | 金属鼓额外 `+10` |
| `song.go` | 104 | 鼓时长固定 `0.1` |
| `dynamics.go` | 42 | kick pitch 固定 `36` |
| `drum.go` | - | 所有 GM 鼓映射（36=kick, 38=snare, 42=hihat）写死 |

## 3. 贝斯硬编码 (bass hardcodes)

| 文件 | 行 | 问题 |
|------|-----|------|
| `song.go` | 146-173 | bass 时长和力度按风格写死 |
| `song.go` | 159 | hiphop bass `root-12` 固定降八度 |
| `song.go` | 169 | pop walking bass 固定 root → third → fifth → root |

## 4. 旋律展开硬编码 (melody expansion hardcodes)

| 文件 | 行 | 问题 |
|------|-----|------|
| `song.go` | 241 | pitch clamp `[21, 108]` |
| `song.go` | 248-252 | metal 旋律 `step*0.6`, velocity `90+10*(bi%2)` |
| `song.go` | 257 | hiphop sync points `[0, 0.5, 1.5, 2.0, 3.0, 3.5]` |
| `song.go` | 264-265 | pop duration `*0.85`, velocity `60+15*(bi%3)` |
| `melody.go` | 34-36 | MaxInterval=7, StepBias=0.7, Gravity=0.4 |
| `melody.go` | 75 | 强拍判定窗口 `<0.5` |
| `melody.go` | 125 | barsPerPhrase = 4 |
| `motif.go` | 246-250 | 音区扩展 `+2/+5/+7`, cap `12` |
| `motif.go` | 261 | syncopation 概率 `0.2` |
| `motif.go` | 275 | anacrusis 偏移 `0.5` |

## 5. 段落和结构硬编码 (structure hardcodes)

| 文件 | 行 | 问题 |
|------|-----|------|
| `song.go` | 78 | sectionDefsForStyle 写死了 4 种风格的段落长度 |
| `transition.go` | 196-206 | BuildSectionProfile 能量值 [0.2,0.4,0.6,0.85,...] |
| `structure.go` | 108 | 段落力度回退公式 `0.5 + energy*0.8` |

## 6. 其他通用硬编码

| 文件 | 行 | 问题 |
|------|-----|------|
| `energy.go` | 63 | 能量权重 `density*0.4 + velocity*0.3 + range*0.3` |
| `groove.go` | 86-87 | humanize `0.05+energy*0.1`, accentBeat `0.6+energy*0.3` |
| `groove.go` | 108-119 | swing 偏移 `0.167/0.083/0.1` |
| `groove.go` | 148 | rubato `0.03` delay, `1.08` stretch |
| `melody.go` | 189 | gravity pull 距离 `>5` |
| `substitution.go` | 45 | 和弦代换率 `tension*0.4` |
| `chord.go` | - | 所有和弦转位规则 |

## 清理策略

1. **阈值类**: 改为读 ComposerDNA 参数，不硬编码 switch
2. **公式类**: 暴露为可调参数（velocity/duration 公式用配置）
3. **音高类**: 使用 MusicDNA 的 key 和 scale 动态计算，不用绝对数字
4. **结构类**: 从 StructureDNA 模板读，不写死
