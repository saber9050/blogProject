# Blog Project

个人博客全栈项目，分为前后端两个独立子项目：

| 子项目 | 目录 | 说明 |
|---|---|---|
| 后端服务 | [blog/](./blog/) | Go + Gin + GORM + MySQL/Redis，提供 RESTful API |
| 前端页面 | [vue/](./vue/) | Vue 3 + TypeScript + Vite，浏览器端 SPA |

## 快速导航

- [后端 README — 技术栈、配置、API 路由、中间件](./blog/README.md)
- [前端 README — 技术栈、目录结构、页面路由、Axios 拦截器](./vue/README.md)

## 项目架构概览

```
┌──────────┐     HTTP/API      ┌──────────┐     ORM      ┌─────────┐
│   Vue    │ ────────────────→  │   Gin    │ ──────────→  │  MySQL  │
│  SPA     │ ←──────────────── │  Backend │ ←──────────  │         │
│ :3000    │     JSON Resp     │ :9527    │     /Redis    │         │
└──────────┘                   └──────────┘               └─────────┘
```

- **前端**：Vue 3 + Pinia + Vue Router + Axios + wangEditor
- **后端**：Go + Gin + GORM + JWT + Redis + MinIO

详情见各子项目 README。
