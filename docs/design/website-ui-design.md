# Agno-Go 官网 UI 设计规范（克制 · 阅读优先 · 无 AI 味）

> 状态：**设计方向草案**。目标：官网（VitePress 文档站 + 落地页）视觉规范。
> 参考：Mintlify（文档站标杆）、Vercel（黑白极简）、Stripe（浅色优雅）、Go 官方风格。
> 原则：**克制、统一、阅读优先**。明确禁止"AI 味"（见第 6 节）。

---

## 1. 设计方向一句话

**白底、深色文字、一个强调色、大留白、无动效装饰** —— 像一个认真写的技术文档，而不是 AI 生成的营销页。

## 2. 禁止清单（AI 味检测器）

以下元素**一律不用**，评审时逐条自查：

| 禁止项 | 典型 AI 味表现 |
|--------|---------------|
| 深色霓虹渐变背景 | 黑底 + 紫色/粉色 glow 光晕 |
| WebGL / canvas 装饰动效 | 粒子漂浮、鼠标跟随、3D 旋转 |
| 随机多色主题 | 每个区块换一种渐变主色 |
| 玻璃拟态滥用 | 大面积 backdrop-blur 毛玻璃卡片 |
| 夸张阴影与圆角 | 大投影、超大圆角气泡感 |
| emoji 做图标 | 🚀✨🔥 当 section 图标 |
| 通用 AI 文案 | "Unleash the power of…"、火箭起飞隐喻 |
| 渐变文字标题 | background-clip: text 的彩虹/霓虹标题 |
| 无意义动效 | 数字滚动、卡片无限漂浮、旋转徽章 |

## 3. 色彩规范（Light 为主，Dark 可后补）

### 主色板（克制黑白灰 + 单一强调色）

```
--bg:          #ffffff            页面背景（纯白）
--surface:     #fafafa            次级表面（代码块底、hover）
--surface-2:   #f5f5f5            更深表面（输入框底）
--text:        #0d0d0d            主文字（近黑，非纯黑）
--text-2:      #333333            次级文字
--text-3:      #666666            弱化文字/说明
--border:      rgba(0,0,0,0.05)   分隔线（5% 黑，极淡）
--border-2:    rgba(0,0,0,0.08)   交互元素边框
--accent:      #00ADD8            Go 青蓝（强调色，唯一彩色）
--accent-deep: #007D9C            强调色深版（hover/文字用）
--accent-bg:   #e6f7fb            强调色浅底（badge/标签）
--code-bg:     #fafafa            代码块背景
```

### 使用纪律

- `--accent` 只用于：CTA 按钮、链接 hover、焦点环、少量强调文字。
- **绝不用彩色做装饰填充**、不做彩色渐变背景、不做彩色卡片。
- 白色区域之间的层级靠 border（5% 黑线）和留白区分，不靠灰底分段。

## 4. 字体规范

```
正文：Inter（400/500/600，无 700+）
代码：Geist Mono（或 JetBrains Mono）
字号：
  展示标题  48px  weight 600  letter-spacing -0.8px
  章节标题  32px  weight 600  letter-spacing -0.5px
  卡片标题  20px  weight 600
  正文      16px  weight 400  line-height 1.7
  小字      14px  weight 400  color #666
  代码      14px  Geist Mono
  标签      uppercase 12px  letter-spacing 0.6px
```

## 5. 布局与组件

### 5.1 页面结构（文档站）

```
┌──────────────────────────────────────────┐
│ Nav：logo | Guide | API | 示例 | 博客 | GitHub│  ← 白底，底边 1px 5% 黑线
├──────────┬───────────────────────────────┤
│ Sidebar  │  正文（最大宽度 ~720px）         │  ← 阅读优先：窄栏正文
│ (Guide   │  h1 → 段落 → 代码块 → 图示      │
│  导航)    │  每节间距 48px+               │
├──────────┴───────────────────────────────┤
│ Footer：文档 | GitHub | License | 社区      │
└──────────────────────────────────────────┘
```

### 5.2 落地页（首页）结构

1. **Hero**：白底。大标题（48px，近黑）+ 一句副标题 + 两个按钮（黑色实心主 CTA + 白底描边次 CTA）。无渐变背景、无 3D。
2. **特性区**：3 列卡片网格。卡片白底 + 5% 黑边框 + 16px 圆角 + 24px 内边距。图标用简单线性 SVG（单色，accent 或近黑）。
3. **代码展示区**：白底 + 左侧代码块（Geist Mono，`#fafafa` 底）。展示真实 Go 代码，不是伪代码。
4. **对比表**：agno-go vs agno vs LangChain vs PydanticAI。朴素表格，边框 5% 黑线。
5. **性能区**：benchmark 图表（静态 SVG 柱状图，单色系）。
6. **CTA 尾区**：白底（或近黑底可选）大标题 + 按钮。不用渐变。
7. **Footer**：朴素，链接列表。

### 5.3 组件细节

| 组件 | 规范 |
|------|------|
| 按钮（主） | 近黑底 #0d0d0d、白字、8px 圆角、hover 透明度 0.9 |
| 按钮（次） | 白底、1px 8% 黑边框、近黑字 |
| 卡片 | 白底、1px 5% 黑边框、16px 圆角、24px 内边距、无阴影或 0.03 极淡 |
| 代码块 | #fafafa 底、1px 5% 边框、8px 圆角、Geist Mono 14px |
| Badge | accent-bg 底 + accent-deep 字、4px 圆角、12px uppercase |
| 表格 | 无斑马纹、仅 5% 黑线分隔、表头 600 |
| 图示 | Mermaid/SVG 线条图：近黑线条 + 白底，强调处用 accent |

### 5.4 文档内图示风格（docs 配图）

- 架构图：**SVG 线条图**（近黑线 + 白底 + accent 点缀），不用 AI 生成图。
- 时序图：Mermaid sequenceDiagram，浅色主题。
- 流程图：Mermaid flowchart，浅色主题。
- 封面/博客头图：Agnes Image 2.1 Flash 生成，但风格限定为"浅色、留白、无霓虹"（提示词里明确禁渐变光效）。

## 6. 风格参考与不参考

**参考**（克制派）：
- Mintlify（文档站标杆：白底、细边框、阅读优先）
- Vercel（黑白极简、精确排版）
- Stripe 浅色页（优雅留白）
- Go 官方站 golang.org（朴素可信）

**不参考**（AI 味重灾区）：
- 各类"AI 创业模板"（深色霓虹、粒子动效、渐变多色）
- Linear/Cursor 式深色 glow 风（好看但辨识度已被 AI 模板玩坏，且不符合阅读优先）

## 7. 落地步骤

1. 先做 VitePress 主题定制：`website/.vitepress/theme/` 覆盖 CSS 变量（色彩/字体/边框）——已有 VitePress 基础，改造量小。
2. 落地页首页：新写 `website/index.md` 对应组件（用 Vue 组件或静态 HTML）。
3. 文档配图：每篇 guide 至少 1 张 SVG/Mermaid 图（见第 5.4）。
4. 首页 + 架构页先做，评审通过后铺开到全部页面。
5. `npm run docs:build` 验证。
