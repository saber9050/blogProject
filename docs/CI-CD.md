# CI/CD 流水线说明

从 `git push` 到线上生效的全自动流程，基于 **GitHub Actions + 阿里云 ACR + 阿里云 ECS**。

## 一、流水线全貌

```text
开发者 git push
        │
        ▼
┌────────────────── CI（每次 push / PR）──────────────────┐
│  backend      go vet → gofmt → go test -race → go build │
│  frontend     npm ci → vue-tsc → vitest → npm run build │
│  integration  起真实 MySQL → 仓储层集成测试（21 个用例） │
│  docker       compose 语法校验 → 两个 Dockerfile 试构建  │
│  security     密钥泄露扫描 → 依赖漏洞扫描                │
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

然后手动放三个东西到 `/opt/blog`：

```bash
# 本地执行
scp docker-compose.prod.yml root@<你的IP>:/opt/blog/
scp .env.example           root@<你的IP>:/opt/blog/.env   # 记得改掉默认密码！

# 后端配置（含数据库密码 / JWT 密钥，不入镜像、不入 git，必须放服务器上）
ssh root@<你的IP> "mkdir -p /opt/blog/configs"
scp blog/configs/config.yaml.example root@<你的IP>:/opt/blog/configs/config.yaml
ssh root@<你的IP> "vi /opt/blog/configs/config.yaml"      # 填真实值
```

> **为什么配置要外挂而不是打进镜像？**
> `blog/.dockerignore` 排除了 `configs/config.yaml`，镜像里只有模板 `config.yaml.example`。
> 敏感配置一旦进镜像，任何能拉到镜像的人都能拿到数据库密码；而且换环境就得重新构建。
> 现在由 `docker-compose` 把宿主机的 `configs/config.yaml` 只读挂载进容器，
> 改配置只需改文件 + `docker compose restart blog-backend`，不用重新构建、不用重新部署。
> 容器网络内的地址（mysql / redis / minio 主机名）由 compose 的环境变量覆盖，
> 所以配置文件里的 `host` 保持占位值即可。

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

这样 PR 不绿就合不进去，CI 才真正成为质量门禁，而不是摆设。

**方式一：网页操作**

> GitHub 界面默认为英文，下面括号里是英文原文，照着找即可。
> 本仓库已经用命令行建好了一条规则，想直接看结果跳到「查看已有规则」。

**A. 查看 / 修改已有规则**

1. 打开仓库首页，点顶部 **Settings**（要有管理员权限才看得到）
2. 左侧菜单 **Code and automation** → **Branches**
3. 页面往下滑到 **Branch protection rules** 区域（在 Rulesets 下方，是旧版入口）
4. 点 `main` 那条规则右侧的 **Edit**
5. 改完拉到底点 **Save changes**

**B. 从零新建一条规则**

1. 仓库 → **Settings** → **Branches**
2. **Branch protection rules** 区域 → **Add branch rule**（有些账号显示为 Add classic branch rule）
3. **Branch name pattern** 填 `main`
4. 勾选 **Require status checks to pass before merging**
5. 勾选 **Require branches to be up to date before merging**
   —— 不加这条，别人能拿过期的旧代码合并进来
6. 在下方搜索框里依次输入并勾选这三个（就是 `ci.yml` 里的 job name）：
   - `后端 Go`
   - `前端 Vue`
   - `镜像可构建性校验`
7. 可选：勾 **Do not allow bypassing the above settings** 表示管理员也不能绕过。
   个人项目建议**不勾**，给自己留个应急通道
8. 页面最下方 **Create** / **Save changes**

**C. 用新版 Rulesets（GitHub 现在的推荐入口，可选）**

路径是 Settings → **Rules** → **Rulesets** → **New ruleset** → **New branch ruleset**。
注意：Rulesets 里要求的状态检查只能从**已经跑过**的检查里选，
所以必须先让 CI 至少成功跑一次，否则下拉框里搜不到 `后端 Go`。
这也是上面 B 方案（旧版入口）更省事的原因——它可以手工填写检查名。

**不要勾的选项**

| 选项 | 为什么别勾 |
|---|---|
| Require a pull request before merging | 会连你自己直接 push main 也一起禁掉，CD 就触发不了了 |
| Require approvals | PR 作者不能 approve 自己的 PR，个人项目等于把自己锁在门外 |
| Require linear history | 需要 rebase 功底，容易把日常提交搞乱 |

**方式二：命令行（已按此配置执行）**

```bash
gh api -X PUT repos/saber9050/blogProject/branches/main/protection \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["后端 Go", "前端 Vue", "镜像可构建性校验"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF
```

查看 / 撤销：

```bash
gh api repos/saber9050/blogProject/branches/main/protection
gh api -X DELETE repos/saber9050/blogProject/branches/main/protection/required_status_checks
```

**怎么验证它真的生效**

1. 从 `develop` 往 `main` 提一个 PR（哪怕只改一个 README 标点）
2. 打开 PR 页面，拉到最下面的合并区
3. CI 还在跑时，你会看到黄色圆点和
   `Some checks haven't completed yet`，Merge 按钮是灰的
4. CI 全绿后变成 `All checks have passed`，按钮才变绿可点
5. 想看反例：临时把某个测试改成必然失败再提 PR，按钮会一直灰着并显示
   `Required checks must pass before merging` —— 这就是门禁在起作用

**两个注意点**

- 首次配置后提第一个 PR 时，检查项会显示「Expected — Waiting for a status to be reported」。
  这是正常的：GitHub 在等这个检查第一次上报。CI 跑完就变绿。
- 本规则只约束**通过 PR 合并**进 main，不阻止你本地 `git push origin main`。
  所以「develop → main 合并后推送 → 触发 CD」的流程不受影响。

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
| `denied: unknown manifest class for application/vnd.oci.empty.v1+json` | Buildx 默认附带 provenance/SBOM 证明清单，阿里云 ACR **个人版**不支持该 OCI 类型 | 已在 `cd.yml` 加 `provenance: false` + `sbom: false`；若自建流水线请务必加上 |
| `denied: requested access to the resource is denied` | ACR 用户名/密码错，或命名空间不存在 | 用 `docker login` 在本机先验证；注意 ACR 密码是独立设置的，**不是**阿里云登录密码 |
| CD 没触发 | `main` 不是默认分支，或 CI 没跑完 | 检查 Actions 里 CI 是否为绿色 |
| 部署作业在「同步编排文件到服务器」失败：`dial tcp ... i/o timeout` | ECS 安全组没放行 22 端口（GitHub Runner 从公网连你） | 安全组入方向加 22 端口；嫌 0.0.0.0/0 太宽可只放行 GitHub Actions 的 IP 段 |
| 同上，报 `unable to authenticate, permission denied (publickey)` | 私钥和服务器上的公钥不匹配，或服务器只设了密码登录 | 本机 `ssh -i 私钥 root@IP echo ok` 能通才说明密钥对；不通就把公钥重新装到服务器 |
| 同上，报 `invalid private key` / `pem` 相关 | Secrets 里私钥粘贴不完整 | 必须包含 `-----BEGIN ... PRIVATE KEY-----` 到 `-----END ... PRIVATE KEY-----` 的完整内容，含头尾行和换行 |
| 同上，报 `scp: ... No such file or directory` | 目标目录不存在 | 先跑 `scripts/setup-server.sh` 或手动 `mkdir -p /opt/blog` |
| `docker pull` 超时 | 服务器没配镜像加速 | 重跑 `setup-server.sh` |
| 后端容器启动即退出，日志 `panic: ... Config File "config" Not Found in "[/app/configs /app]"` | 服务器上缺 `/opt/blog/configs/config.yaml`（该文件不入镜像，必须外挂） | 见第二节第 2 步；`deploy.sh` 已加前置校验，会在拉取镜像前就失败并提示如何补 |
| 部署后 502 | 后端健康检查未通过 | `docker logs blog-backend` 看启动日志；`deploy.sh` 回滚前会自动打印最近 50 行 |
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
8. **镜像冒烟测试**：CD 在推送后、部署前先 `docker run` 自检镜像（二进制存在、配置文件已打包、前端产物存在），把"构建成功但一启动就崩"挡在上线之前。
9. **配置与镜像分离**（12-Factor）：敏感配置不进镜像、不进 git，通过只读挂载注入，一份镜像可跑多环境，改配置无需重新构建。

## 七、可以继续加的

- **Codecov**：把覆盖率报告传上去，PR 里显示覆盖率变化
- **Dependabot**：依赖漏洞自动提 PR
- **多环境**：加 `staging` 环境，先灰度再生产
- **通知**：部署结果推送到钉钉/飞书机器人
- **数据库迁移**：接 `golang-migrate`，把 schema 变更也纳入流水线
