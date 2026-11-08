# Agent Rules

本文件为 AI Agent 提供项目开发指南，Agent 在编写代码前**必须先阅读本文件**。

---

## 🚨 强制执行规则（优先级最高）

在编写任何代码前，必须完成以下步骤：

### 0. 项目位置确认（最高优先级）

> ⚠️ **写代码前必须先确认正确的项目位置，这是最常见的错误来源！**

#### 必须执行的检查：

1. **识别项目根目录**
   - 查找 `package.json` 所在位置，该位置即为项目根目录
   - 项目根目录通常包含：`package.json`、`src/`、`node_modules/`、`vite.config.js` 等
   - 如果工作区根目录与项目根目录不同，必须明确区分

2. **路径对齐原则**
   - 所有文件操作路径必须基于**项目根目录**，而非工作区根目录
   - 示例：若项目位于 `aiide-web-prototype/` 子目录
     - ✅ 正确：`aiide-web-prototype/src/pages/Home/index.jsx`
     - ❌ 错误：`src/pages/Home/index.jsx`（在工作区根目录创建）

3. **操作前验证**
   - 创建/修改文件前，先用 `ls` 确认目标目录存在且路径正确
   - 确保新文件的父目录路径与现有项目结构一致

#### 常见错误场景：
| 场景 | 错误做法 | 正确做法 |
|-----|---------|---------|
| 项目在子目录 `my-app/` | 在根目录创建 `src/pages/` | 在 `my-app/src/pages/` 创建 |
| 多项目工作区 | 混淆不同项目的 src 目录 | 明确指定完整路径前缀 |

> 📌 **总则**：先定位项目根目录，再操作文件。路径错误 = 任务失败。

### 1. 配置审查（先检查再编码）

#### 必须读取并分析：

**1. tailwind.config.js**
- 检查并记录：colors / keyframes / animation
- 若缺少 primary / secondary / accent / foreground 系列 → **必须补充完整色系**
- 若已存在主题色 → **增量开发不可大幅修改，仅可按需扩展或使用 tailwind 的默认调色板**
- 配色要求：对比度清晰、避免紫色为主调、统一风格美术方向

**颜色体系规范：**
| 名称 | 用途 |
|---|---|
| primary | 主色（主要操作按钮背景） |
| secondary | 次色（次级按钮背景） |
| accent | 强调色（提示/高亮） |
| foreground | 主文本 |
| muted-foreground | 次文本 |
| border | 边框 |
| muted | 浅色背景 |

**2. src/index.css**
- 记录所有 `@layer components` 中的预定义 class

**3. package.json**
- 检查依赖是否已包含将要使用的组件库/动画库
- 阅读 `agentGuide` 词条了解各个库的使用情况，并结合 tailwind.config.js 使用

---

### ⛔ 代码编写禁止项
- ❌ 内联硬编码颜色（例：`style={{ color: "#9900ff" }}`）
- ❌ 不存在的色阶（例：`bg-neutral-450`）
- ❌ 未列入 tailwind.config.js 配置的 font / animation

### ✅ 正确使用规范
- ✔ 使用 Tailwind 已定义颜色：`bg-neutral-500`
- ✔ 使用 index.css 组件类：`btn-primary-500`
- ✔ 允许使用 tailwind 的默认调色板，如 `pink-500`
- ✔ 使用已声明动画：`animate-fade-in`
- ✔ 允许在 tailwind.config.js 中新增自定义动画（如 `spin-soft`），并在代码中通过 `animate-spin-soft` 使用

> 📌 **总则**：先对齐色系配置再编码。未设定颜色 → 必须补设；已设定颜色 → 稳定优先，不随意变更

---

## 📋 开发铁律（违反即失败）

### 1. 组件管理
- 删除所有无关的模板默认组件
- 保持组件架构完整性

### 2. 样式使用严格规则
- **优先**使用 tailwind.config.js 中定义的颜色组（primary/secondary/accent）以呈现主题色调
- **允许**使用 `pink-500` 此等 tailwind 的默认颜色组作为关键配色
- **建议**优先使用 index.css 中的预定义类（btn-primary、btn-secondary 等）
- **禁止**任何硬编码颜色值
- **推荐**当设计需要视觉层级、氛围渲染或区块分隔时，优先考虑使用渐变背景：
  - 方向控制：`bg-gradient-to-r` / `bg-gradient-to-b` / `bg-gradient-to-tr` 等
  - 色组组合示例：`from-secondary-700 to-blue-500`，`from-secondary-500 via-pink-500 to-blue-300`
  - 典型使用场景：Hero Banner、Section 背景、页脚或分隔区域等

### 3. 配色决策流程
```
需要颜色 → 检查 config 是否有对应语义 → 有则使用 / 无则修改 config → 使用新配置
```
> 绝不允许跳过 config 直接使用颜色

### 4. 图片资源
- 使用 web_search 获取远程图片链接，图片内容必须与产品名称、功能特性精准匹配，避免泛化配图
- 优先选择知乎、主流媒体、官网、Unsplash、Pexels 等稳定来源的高清图片
- 在实装图片到代码中之前，根据当前运行环境验证链接可用性（如 curl / wget / fetch 等），若失败则更换关键词重试，最多 3 次
- 所有图片默认用于「界面展示」，单张图片建议小于 1MB，最大不超过 3MB

### 5. 动画策略
- 简单动画：使用 tailwind 配置的 `animate-{fade-in|slide-up|scale-in}`
- 复杂交互：使用 Framer Motion
- 大型动效：按需引入 gsap/lottie-web/animate.css
- 图表组件：按情况使用 echarts 与 recharts

### 6. 路由配置
- **同一项目内**若需要建立多个页面进行跳转/导航，必须使用已配置的 react-router
- 多页面导航必须使用哈希路由（HashRouter），否则无法部署
- 若非同一项目（如：购物平台官网和物品详情页为同一项目，而技术中台页面和电商购物则不是同一项目），请在根目录下新建文件夹，按照 workflow 流程规划

### 7. 依赖管理
- 优先使用已安装依赖
- 按需添加可选依赖（gsap、lottie-web、animate.css）

### 8. 文档参考
- 开发前必读 README.md

---

## ✅ 代码生成后自检清单

生成代码后，必须在内部执行以下检查：
- [ ] 所有颜色是否来自 tailwind 默认调色板或 tailwind.config.js？
- [ ] 是否使用了 index.css 中的预定义类？
- [ ] 是否删除了无关的模板组件？
- [ ] 是否有任何硬编码样式？

---

## 核心原则

1. **动效优先**：生成任何前端界面时，必须主动评估页面结构、内容类型与视觉需求，并在可行的场景下借鉴已有的实现方式，灵活组合使用多种动效库，让界面更具吸引力、更有生命力，能用动效库就用动效库。

2. **样式规范**：凡包含颜色、字体、阴影、spacing、动画 keyframes 的设计需求，均需优先通过 Tailwind 配置（`tailwind.config.js`）扩展实现，而非临时样式或硬编码。

3. **对比度原则**：文字/组件/icon 颜色与背景必须保持强对比度，避免白底白字、白底白图、黑底黑字、黑底黑图等可读性问题。

## 动效设计规范

1. **动效必须服务体验，而不是喧宾夺主**
2. **不同动效库可组合使用，每个模块选择最适合的库**
3. **页面动效需遵循视觉风格**：极简风格用轻动效，营销页可更强烈
4. **关键区域应优先设计动效**：Hero 区域、卡片 hover、CTA 按钮、滚动交互等

---

## 目录结构

```
src/
├── components/     # 公共组件（多个页面复用的组件）
├── pages/          # 页面组件
│   └── PageName/
│       ├── index.jsx           # 页面入口
│       └── components/         # 页面私有组件（仅该页面使用）
├── utils/          # 工具函数
├── App.jsx         # 主应用（路由配置）
├── main.jsx        # 入口文件
└── index.css       # 全局样式
```

## 组件化开发规范

> ⛔ **严禁将所有代码写在单个 index.jsx 文件中**，必须进行合理的组件拆分

1. **组件拆分原则**
   - 页面必须拆分为多个独立组件，每个组件职责单一
   - 单个组件代码建议不超过 200 行，超过时应考虑拆分
   - 典型拆分示例：Header、Hero、Features、Footer 等应为独立组件

2. **组件存放位置**
   - 仅当前页面使用的组件 → `src/pages/PageName/components/`
   - 多个页面复用的组件 → `src/components/`

3. **组件文件命名**
   - 使用 PascalCase：`UserCard.jsx`、`DataChart.jsx`
   - 复杂组件可用文件夹：`components/UserCard/index.jsx`

4. **新建页面流程**
   - 在 `src/pages/` 下创建页面目录及 `components/` 子目录
   - 页面入口 `index.jsx` 仅负责组合各子组件
   - 在 `App.jsx` 中添加对应路由

---

## 库使用指南

### 图标
| 库 | 说明 |
|---|------|
| `lucide-react` | 图标库，提供 1000+ 简洁线性图标，用法：`import { IconName } from 'lucide-react'` |

### 工具类
| 库 | 说明 |
|---|------|
| `clsx` | 条件拼接 className 的工具，用法：`clsx('base', { 'active': isActive })` |
| `tailwind-merge` | 合并 Tailwind 类名并自动处理冲突，配合 clsx 使用 |

### 动画库（按场景选择）

| 库 | 适用场景 |
|---|---------|
| `framer-motion` | React 动画库，适用于：组件进入/退出动画、手势交互、布局动画、复杂编排动画 |
| `animate.css` | CSS 动画库，适用于：简单的预设动画效果，通过添加类名使用，如 `animate__fadeIn` |
| `gsap` | 专业级动画库，适用于：复杂时间轴动画、ScrollTrigger 滚动动画、高性能动画需求 |
| `lottie-web` | Lottie 动画播放器，适用于：播放 AE 导出的 JSON 动画文件，常用于加载动画、图标动画 |

### 图表库（按场景选择）

| 库 | 适用场景 |
|---|---------|
| `echarts` + `echarts-for-react` | 百度图表库，适用于：数据可视化大屏、深色科技风主题、复杂图表（仪表盘、水球图、地图、桑基图等）、丰富动效 |
| `recharts` | React 图表库，适用于：一两个需要使用简单图表需求的地方、React 声明式风格、轻量级场景、基础柱状图/折线图/饼图 |
