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
│   │   ├── NavBar.vue         # 顶部导航栏（含"随便看看"随机文章）
│   │   └── ToastContainer.vue # Toast 消息容器
│   ├── views/
│   │   ├── HomeView.vue           # 首页（文章列表 + 侧边栏）
│   │   ├── LoginView.vue          # 登录
│   │   ├── RegisterView.vue       # 注册
│   │   ├── ForgotPasswordView.vue # 忘记密码
│   │   ├── ArticleDetailView.vue  # 文章详情
│   │   ├── AdminView.vue          # 管理后台
│   │   ├── ArticleEditorView.vue  # 文章编辑器（新建/编辑）
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
| `/` | 首页 | 文章列表 + 侧边栏 |
| `/login` | 登录 | 账号密码登录或邮箱验证码登录 |
| `/register` | 注册 | 新用户注册 |
| `/forgot-password` | 忘记密码 | 密码重置 |
| `/article/:id` | 文章详情 | 查看完整文章（含评论、点赞） |
| `/admin` | 管理后台 | 文章/分类/评论/标签/用户管理（需管理员） |
| `/admin/article/new` | 新建文章 | 管理员写文章 |
| `/admin/article/:id/edit` | 编辑文章 | 管理员编辑已有文章 |
| `/profile` | 个人中心 | 个人信息修改、头像更换、邮箱变更 |
| `/about` | 关于 | 关于页面 |
| `/*`（未匹配） | 重定向 | 所有未匹配路径重定向到首页 |

## 主要功能

- 文章列表浏览（支持分类筛选、标签多选、关键词搜索、最新/热门排序）
- 文章全文展示 + 嵌套评论/回复
- 文章点赞/取消点赞
- 文章统计（总文章数、总浏览量、总点赞数）
- 随机一篇文章（导航栏"随便看看"）
- 分类/标签溢出时自动折叠，支持展开/收起
- 用户注册/登录/密码重置（邮箱验证码）
- 后台管理：文章、分类、评论、标签、用户 CRUD
- 富文本编辑器（wangEditor）撰写文章
- 分类一键转移
- 个人资料编辑 + 头像上传 + 邮箱修改
- Toast 消息通知（success / error / info）

## Axios 拦截器

### 请求拦截器
- 自动从 `localStorage` 读取 `token`（即 access_token），注入 `Authorization: Bearer` 请求头

### 响应拦截器 — 双 Token 自动刷新

采用 **双 Token 机制**（Access Token + Refresh Token）：

| 机制 | 说明 |
|---|---|
| Access Token | JWT，15 分钟有效期，内含 `tid` 字段（即 refresh token） |
| Refresh Token | 随机字符串，同时是会话标识符，默认 3 天有效期，存 Redis |
| 刷新触发 | 任意接口返回 401 时自动调用 `/auth/refresh` |

**刷新流程**：
1. 请求返回 401 → 检查是否为 `/auth/refresh` 接口自身（是则直接拒绝，避免循环）
2. 若已有刷新请求进行中 → 将请求排队等待复用同一个刷新结果
3. 否则发起刷新请求（带过期 access token），服务端解析 `tid` 验证并轮换
4. 刷新成功 → 更新本地 `token` + 唤醒排队队列，重放原请求
5. 刷新失败 → 清除本地凭证，弹出 `登录信息已过期，请重新登录` 提示，跳转登录页

**并发安全**：
- `isRefreshing` 互斥锁确保同一时间只有一个刷新请求
- `failedQueue` 排队机制，多个并发 401 请求共享同一个刷新结果

## 后端 API 代理

开发环境下，`/api` 开头的请求通过 Vite 代理转发到后端：

```ts
// vite.config.ts
server: {
  host: '0.0.0.0',
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:9527',
      changeOrigin: true,
      rewrite: (path) => path
    }
  }
}
```

生产环境需在 Nginx 或后端配置同源策略。
