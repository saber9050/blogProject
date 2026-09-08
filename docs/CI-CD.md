# CI/CD 流水线说明

从 `git push` 到线上生效的全自动流程，基于 **GitHub Actions + 阿里云 ACR + 阿里云 ECS**。

## 一、流水线全貌

```text
开发者 git push
        │
        ▼
┌────────────────── CI（每次 push / PR）──────────────────┐
│  backend    go vet → gofmt → go test -race → go build   │
│  frontend   npm ci → vue-tsc → vitest → npm run build   │
│  docker     compose 语法校验 → 两个 Dockerfile 试构建   │
└─────────────────────────────────────────────────────────┘
        │  仅 main 分支 且 CI 全绿
        ▼
┌────────────────── CD（自动 / 手动）────────────────────┐
│  1. 解析 commit sha 作为镜像 tag（可追溯、可回滚）      │
│  2. 在 GitHub Runner 上构建镜像                          │
│  3. 推送到阿里云容器镜像服务 ACR                         │
│  4. SCP 同步 compose 文件与部署脚本到 ECS                │
│  5. SSH 执行 deploy.sh：pull → up -d → 健康检查          │
│  6. 健康检查失败 → 自动回滚上一个 tag                    │
└─────────────────────────────────────────────────────────┘
        │
        ▼
   https://你的域名或IP  更新完成
```

**为什么构建放在 GitHub Runner 而不是服务器？**
你的 ECS 是 2vCPU 4GiB，同时编译 Go + Vue 会吃满内存、拖慢线上服务。构建产物（镜像）推送到 ACR 后，ECS 只需 `docker pull`，几秒完成，且 ACR 与 ECS 同地域可走内网、免公网流量费。

## 二、一次性配置（约 30 分钟）

### 1. 开通阿里云容器镜像服务 ACR

1. 控制台 → 容器镜像服务 ACR → 创建**个人版实例**（免费）
2. 设置 Registry 登录密码（**不是**阿里云账号密码）
3. 创建命名空间，例如 `saber9050`
4. 记录：
   -  registry 地址：`registry.cn-hangzhou.aliyuncs.com`（按你的地域）
   - 命名空间：`saber9050`
   - 用户名：阿里云账号全名
   - 密码：上一步设置的密码

### 2. 服务器初始化

SSH 登录 ECS，执行：

```bash
bash scripts/setup-server.sh
```

它会安装 Docker、配置国内镜像加速、创建 2GB Swap、建好 `/opt/blog`。

然后手动放两个文件到 `/opt/blog`：

```bash
# 本地执行
scp docker-compose.prod.yml root@<你的IP>:/opt/blog/
scp .env.example           root@<你的IP>:/opt/blog/.env   # 记得改掉默认密码！
```

安全组放行端口：**80、9527**（9000/9001 建议只对自家 IP 开放，3306/6379 绝不要开公网）。

首次启动（会拉 gemma3:270m 模型，耐心等几分钟）：

```bash
cd /opt/blog && docker compose -f docker-compose.prod.yml up -d
```

### 3. 配置 SSH 免密

```bash
# 本地生成专用密钥（不要复用已有的）
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/blog_deploy
# 把公钥放到服务器
ssh-copy-id -i ~/.ssh/blog_deploy.pub root@<你的IP>
# 复制私钥内容，稍后填到 GitHub Secrets
cat ~/.ssh/blog_deploy
```

### 4. 填写 GitHub Secrets

仓库 → Settings → Secrets and variables → Actions → New repository secret：

| Secret | 示例值 | 说明 |
|---|---|---|
| `ACR_REGISTRY` | `registry.cn-hangzhou.aliyuncs.com` | ACR 地域地址 |
| `ACR_NAMESPACE` | `saber9050` | 命名空间 |
| `ACR_USERNAME` | `你的阿里云账号` | 登录用户名 |
| `ACR_PASSWORD` | `***` | ACR 独立密码 |
| `ECS_HOST` | `47.xxx.xxx.xxx` | 服务器公网 IP |
| `ECS_USER` | `root` | 登录用户 |
| `ECS_SSH_KEY` | `-----BEGIN OPENSSH...` | 上一步的**私钥全文** |
| `ECS_DEPLOY_PATH` | `/opt/blog` | 部署目录 |

### 5. 保护 main 分支（强烈建议）

Settings → Branches → Add rule → Branch name: `main`
勾选 **Require status checks to pass before merging**，选中 `后端 Go`、`前端 Vue`、`镜像可构建性校验`。

这样 PR 不绿就合不进去，CI 才真正成为质量门禁，而不是摆设。

## 三、日常使用

```bash
# 开发流程
git switch develop
# ... 写代码 ...
git commit -m "feat: 新增文章搜索"
git push origin develop        # 触发 CI，不部署

# 准备上线
git switch main && git merge develop
git push origin main           # CI 通过 → CD 自动部署
```

手动触发部署：Actions → CD → Run workflow。

查看部署的是哪个版本：

```bash
ssh root@<你的IP> "cat /opt/blog/.env | grep IMAGE_TAG"
docker ps --format '{{.Names}} {{.Image}}'
```

## 四、回滚

**自动**：健康检查 90 秒内没通过，`deploy.sh` 会把 `IMAGE_TAG` 改回上一个版本并重启容器。

**手动**：

```bash
ssh root@<你的IP>
cd /opt/blog
sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=上一个sha/' .env
docker compose -f docker-compose.prod.yml up -d blog-backend blog-frontend
```

## 五、常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| CD 没触发 | `main` 不是默认分支，或 CI 没跑完 | 检查 Actions 里 CI 是否为绿色 |
| `docker pull` 超时 | 服务器没配镜像加速 | 重跑 `setup-server.sh` |
| 部署后 502 | 后端健康检查未通过 | `docker logs blog-backend` 看启动日志 |
| 容器被 OOM 杀掉 | 内存超限 | `docker inspect <容器> \| grep -i oom`，调低 `mem_limit` 或加 Swap |
| 磁盘满了 | 旧镜像堆积 | `docker image prune -a -f`（保留在用镜像） |
| gofmt 检查失败 | 本地没格式化 | `cd blog && gofmt -w .` |

## 六、这套流水线能写进简历的技术点

面试时按"问题 → 方案 → 收益"讲，比罗列工具名有力得多：

1. **质量门禁**：`workflow_run` 让 CD 只在 CI 全绿后触发；main 分支保护强制 PR 通过检查。
2. **不可变制品**：镜像 tag 用 commit sha，构建一次、多环境复用，杜绝"我本地是好的"。
3. **构建与部署分离**：重编译放在 GitHub Runner，4GiB 的小机器只做 pull，把部署时间从分钟级压到秒级，且避免编译抢占线上资源。
4. **零停机与自动回滚**：`up -d` 滚动替换容器，健康检查失败自动切回上一版本——这是 CD 的核心价值，不是"能自动跑脚本"就叫 CD。
5. **资源治理**：compose 里给每个服务设 `mem_limit`，把有限内存按优先级切分；配 swap 兜底。
6. **缓存优化**：`npm ci`、`setup-go` 依赖缓存、Buildx GHA 缓存，二次构建从数分钟降到几十秒。
7. **可观测性**：部署脚本输出每个服务的状态表，出问题时能一眼定位。

## 七、可以继续加的

- **Codecov**：把覆盖率报告传上去，PR 里显示覆盖率变化
- **Dependabot**：依赖漏洞自动提 PR
- **多环境**：加 `staging` 环境，先灰度再生产
- **通知**：部署结果推送到钉钉/飞书机器人
- **数据库迁移**：接 `golang-migrate`，把 schema 变更也纳入流水线
