#!/usr/bin/env bash
# ============================================================
# 部署脚本（在阿里云 ECS 上执行，由 GitHub Actions 通过 SSH 调用）
#
# 流程：
#   1. 登录 ACR
#   2. 拉取新镜像
#   3. 滚动更新应用容器（数据库等有状态服务不动）
#   4. 健康检查
#   5. 失败自动回滚到上一个镜像 tag
#
# 用法：
#   IMAGE_TAG=<git sha> ./deploy.sh [tag]
# ============================================================
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/blog}"
COMPOSE_FILE="docker-compose.prod.yml"
IMAGE_TAG="${1:-${IMAGE_TAG:-latest}}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"

log()  { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
fail() { log "错误：$*"; exit 1; }

cd "$DEPLOY_DIR" || fail "部署目录不存在：$DEPLOY_DIR"
[ -f "$COMPOSE_FILE" ] || fail "缺少 $COMPOSE_FILE"
[ -f ".env" ] || fail "缺少 .env（请先复制 .env.example 并填写）"

# 后端配置文件：含数据库密码等密钥，不入镜像，必须存在于宿主机
# 缺失时后端会 panic: Config File "config" Not Found —— 提前拦截，避免走完整个部署再回滚
if [ ! -f "configs/config.yaml" ]; then
  fail "缺少 configs/config.yaml。请在 $DEPLOY_DIR 下创建：
    mkdir -p $DEPLOY_DIR/configs
    把 blog/configs/config.yaml.example 复制为 configs/config.yaml 并填好真实值
    （mysql/redis/minio 的 host 保持 127.0.0.1 即可，会被 compose 里的环境变量覆盖）"
fi

# 记录当前版本，用于回滚
PREVIOUS_TAG=$(grep -E '^IMAGE_TAG=' .env 2>/dev/null | cut -d= -f2 || echo "latest")
log "当前版本：$PREVIOUS_TAG → 目标版本：$IMAGE_TAG"

# ---------- 1. 登录镜像仓库 ----------
if [ -n "${ACR_USERNAME:-}" ] && [ -n "${ACR_PASSWORD:-}" ]; then
  log "登录镜像仓库 ${ACR_REGISTRY}"
  echo "$ACR_PASSWORD" | docker login "$ACR_REGISTRY" \
    --username "$ACR_USERNAME" --password-stdin
fi

# ---------- 2. 拉取新镜像 ----------
log "拉取新镜像"
IMAGE_TAG="$IMAGE_TAG" docker compose -f "$COMPOSE_FILE" pull blog-backend blog-frontend

# ---------- 3. 更新容器 ----------
log "滚动更新应用容器"
IMAGE_TAG="$IMAGE_TAG" docker compose -f "$COMPOSE_FILE" up -d --remove-orphans \
  blog-backend blog-frontend

# ---------- 4. 健康检查 ----------
check_health() {
  local name=$1 url=$2
  local waited=0
  while [ "$waited" -lt "$HEALTH_TIMEOUT" ]; do
    if curl -fsS --max-time 5 "$url" >/dev/null 2>&1; then
      log "$name 健康检查通过（${waited}s）"
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  return 1
}

if check_health "后端" "http://127.0.0.1:9527/api/v1/health" \
   && check_health "前端" "http://127.0.0.1/"; then
  log "部署成功：$IMAGE_TAG"
else
  log "健康检查失败，开始回滚到 $PREVIOUS_TAG"
  # 打印后端日志，方便直接在 Actions 里定位崩溃原因
  log "----- blog-backend 最近日志 -----"
  docker compose -f "$COMPOSE_FILE" logs --tail=50 blog-backend 2>&1 || true
  log "---------------------------------"
  IMAGE_TAG="$PREVIOUS_TAG" docker compose -f "$COMPOSE_FILE" up -d \
    blog-backend blog-frontend
  fail "部署失败并已回滚"
fi

# ---------- 5. 写入新版本并清理 ----------
if grep -q '^IMAGE_TAG=' .env; then
  sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$IMAGE_TAG/" .env
else
  echo "IMAGE_TAG=$IMAGE_TAG" >> .env
fi

# 清理悬空镜像，避免 60G 系统盘被撑满（保留最近 3 个版本依赖的层）
docker image prune -f --filter "until=72h" >/dev/null 2>&1 || true

log "完成，当前版本：$IMAGE_TAG"
docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Image}}"
