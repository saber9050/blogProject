# Blog-Frontend - 前端项目

基于 Vue 3 + TypeScript + Vite 的个人博客前端。

## 技术栈

| 组件 | 技术 |
|---|---|
| 框架 | Vue 3 (Composition API, `<script setup>`) |
| 语言 | TypeScript |
| 构建 | Vite 5 |
| 路由 | Vue Router 4 (History 模式) |
| 状态管理 | Pinia |
| HTTP | Axios |
| 富文本编辑器 | wangEditor 5 |
| UI | 纯手写 CSS + CSS 变量（无组件库） |

## 目录结构

```
vue/
├── src/
│   ├── main.ts                # 应用入口
│   ├── App.vue                # 根组件
│   ├── assets/
│   │   └── main.css           # 全局样式 + CSS 变量
│   ├── components/
│   │   ├── AuthShell.vue      # 认证页面外壳
│   │   ├── NavBar.vue         # 顶部导航栏
│   │   └── ToastContainer.vue # Toast 消息容器
│   ├── views/
│   │   ├── HomeView.vue           # 首页
│   │   ├── LoginView.vue          # 登录
│   │   ├── RegisterView.vue       # 注册
│   │   ├── ForgotPasswordView.vue # 忘记密码
│   │   ├── ArticleDetailView.vue  # 文章详情
│   │   ├── AdminView.vue          # 管理后台
│   │   ├── ProfileView.vue        # 个人中心
│   │   └── AboutView.vue          # 关于
│   ├── router/
│   │   └── index.ts           # 路由配置
│   ├── stores/
│   │   └── auth.ts            # 认证状态 (Pinia)
│   ├── api/
│   │   └── index.ts           # Axios 实例 + 拦截器
│   └── utils/
│       └── toast.ts           # Toast 通知工具
├── index.html
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 快速开始

### 前置条件

- Node.js 18+
- npm 9+

### 安装

```bash
npm install
```

### 运行开发服务器

```bash
npm run dev
```

启动在 `http://localhost:3000`，API 请求自动代理到 `http://localhost:9527`（后端服务）。

### 构建

```bash
npm run build
```

构建产物输出到 `dist/` 目录。

## 页面路由

| 路径 | 页面 | 说明 |
|---|---|---|
| `/` | 首页 | 文章列表 |
| `/login` | 登录 | 账号密码登录 |
| `/register` | 注册 | 新用户注册 |
| `/forgot-password` | 忘记密码 | 密码重置 |
| `/article/:id` | 文章详情 | 查看完整文章 |
| `/admin` | 管理后台 | 文章/分类/评论/标签管理（需管理员） |
| `/profile` | 个人中心 | 个人信息修改、邮箱变更 |
| `/about` | 关于 | 关于页面 |

## 主要功能

- 文章列表浏览和全文展示
- 用户注册/登录/密码重置
- 后台管理：文章、分类、评论、标签 CRUD
- 富文本编辑器（wangEditor）撰写文章
- 分类一键转移
- 个人资料编辑 + 邮箱修改
- Toast 消息通知

## 后端 API 代理

开发环境下，`/api` 开头的请求通过 Vite 代理转发到后端：

```ts
// vite.config.ts
proxy: {
  '/api': {
    target: 'http://localhost:9527',
    changeOrigin: true
  }
}
```

生产环境需在 Nginx 或后端配置同源策略。
