# React + Vite + Tailwind CSS Template

🎨 **现代化前端开发模板**

这是一个精简的 React + Vite + Tailwind CSS 模板，开箱即用，提供了基础的开发环境配置。

## ✨ 特性

- ⚡️ **Vite** - 极速开发体验
- ⚛️ **React 18** - 最新 React 特性
- 🎨 **Tailwind CSS** - 原子化 CSS 框架
- 🎭 **Framer Motion** - 流畅的动画效果
- 🎯 **Lucide React** - 现代图标库
- 📦 **预配置路径别名** - 便捷的模块导入
- 🎪 **标准配色系统** - Primary / Accent / Neutral

## 📦 安装

```bash
npm install
```

## 🚀 开发

```bash
npm run dev
```

访问 `http://localhost:8080`

## 🏗️ 构建

```bash
npm run build
```

## 📁 项目结构

```
src/
├── components/       # 公共组件目录
├── pages/           # 页面目录
│   └── Home/        # 首页模块
│       ├── index.jsx      # 首页主文件
│       └── components/     # 首页专属组件
├── utils/           # 工具函数
│   └── cn.js        # className 合并工具
├── App.jsx          # 根组件
├── main.jsx         # 入口文件
└── index.css        # 全局样式
```

## 🔍 代码规范（默认关闭）

项目使用 ESLint 进行代码质量检查，基于 **@ecomfe/eslint-config** 规范（百度前端团队代码规范），默认是关闭的。

### 运行 Lint 检查

```bash
# 检查代码
npm run lint

# 自动修复可修复的问题
npm run lint:fix
```

### 主要规范要点

- **缩进**：4 个空格（JavaScript）+ 4 个空格（JSX）
- **未使用变量**：报错级别（`no-unused-vars: error`）
- **对象/数组空格**：紧凑风格，不允许空格（`{a: 1}` ✅ / `{ a: 1 }` ❌）
- **React Hooks**：强制遵循 Hooks 规则
- **JSX 文件扩展名**：`.js`, `.jsx`, `.tsx` 均可

### 自定义规则

如需调整规则，在 [.eslintrc.json](.eslintrc.json) 中添加 `rules` 字段：

```json
{
  "rules": {
    "no-unused-vars": "warn",  // 改为警告
    "indent": ["error", 2]     // 改为 2 空格缩进
  }
}
```

> 💡 提示：在 VSCode 中安装 ESLint 插件可实时查看代码问题

## 🎨 配色系统

模板提供标准化的配色结构，在 `tailwind.config.js` 中定义：

### 色彩层级（各10级：50-900）
- **primary** - 主色调，用于主要按钮、品牌色（如 `bg-primary-500`）
- **secondary** - 次要色，用于次要按钮、辅助元素（如 `bg-secondary-100`）
- **accent** - 强调色，用于高亮、提示等（如 `text-accent-600`）

### 功能性色彩（单色值）
- **foreground** - 深色文本，用于主要内容（如 `text-foreground`）
- **muted-foreground** - 中性文本，用于次要说明（如 `text-muted-foreground`）
- **border** - 边框颜色（如 `border-border`）
- **muted** - 浅色背景（如 `bg-muted`）

> 💡 所有颜色值需在 `tailwind.config.js` 中根据项目需求自定义配置

## 📚 可选动效库

模板已在 `package.json` 中配置以下可选依赖：

- `gsap` - 强大的动画库
- `lottie-web` - Lottie 动画支持
- `animate.css` - CSS 动画库

根据需要安装：

```bash
npm install gsap lottie-web animate.css
```

## 🛠️ 路径别名

已配置以下路径别名：

- `@` → `src/`
- `@components` → `src/components/`
- `@pages` → `src/pages/`
- `@styles` → `src/styles/`
- `@utils` → `src/utils/`

## 📖 使用建议

1. **组件命名** - 使用 PascalCase（如 `Button.jsx`）
2. **样式优先级** - Tailwind > CSS Modules > 全局 CSS
3. **动效使用** - Framer Motion 用于组件级，GSAP 用于页面级
4. **响应式设计** - 优先使用 Tailwind 断点（sm/md/lg/xl/2xl）

## 📝 License

MIT
