#!/usr/bin/env bash
# ============================================================
# 阿里云 ECS 初始化脚本（只需在服务器上执行一次）
#
# 用法：
#   curl -fsSL <raw-url>/setup-server.sh | bash
#   或：bash setup-server.sh
# ============================================================
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/blog}"

log() { echo "==> [$(date '+%H:%M:%S')] $*"; }

# ---------- 1. 安装 Docker ----------
if command -v docker >/dev/null 2>&1; then
  log "Docker 已安装，跳过"
else
  log "安装 Docker"
  if [ -f /etc/alinux-release ] || [ -f /etc/centos-release ] || [ -f /etc/redhat-release ]; then
    # 阿里云 Linux / CentOS：使用阿里云镜像源
    curl -fsSL https://get.docker.com | bash -s docker --mirror Aliyun
  else
    curl -fsSL https://get.docker.com | bash -s docker --mirror Aliyun
  fi
  systemctl enable --now docker
fi

# ---------- 2. 配置镜像加速（解决国内拉不动 Docker Hub） ----------
log "配置 Docker 镜像加速"
mkdir -p /etc/docker
cat > /etc/docker/daemon.json <<'EOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.cn",
    "https://docker.nju.edu.cn"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "20m",
    "max-file": "3"
  },
  "live-restore": true
}
EOF
systemctl daemon-reload
systemctl restart docker

# ---------- 3. 添加 Swap（4GiB 小内存机器必备） ----------
if swapon --show | grep -q '/swapfile'; then
  log "Swap 已存在，跳过"
else
  log "创建 2GB Swap"
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile >/dev/null
  swapon /swapfile
  grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
  # 降低换页倾向，避免正常使用就触发 swap
  sysctl -w vm.swappiness=10 >/dev/null
  grep -q 'vm.swappiness' /etc/sysctl.conf || echo 'vm.swappiness=10' >> /etc/sysctl.conf
fi

# ---------- 4. 创建部署目录 ----------
log "创建部署目录 $DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR/configs"
cd "$DEPLOY_DIR"

# ---------- 5. 提示 ----------
cat <<EOF

$(echo -e '\033[0;32m')初始化完成$(echo -e '\033[0m')

接下来请手动完成 4 件事：

  1) 把仓库中的 docker-compose.prod.yml 和 .env 放到 $DEPLOY_DIR
       scp docker-compose.prod.yml root@<你的IP>:$DEPLOY_DIR/
       scp .env                    root@<你的IP>:$DEPLOY_DIR/

  1.1) 放后端配置文件（密钥不进镜像，必须放在服务器上）
       scp blog/configs/config.yaml.example root@<你的IP>:$DEPLOY_DIR/configs/config.yaml
       然后 vi $DEPLOY_DIR/configs/config.yaml 填真实值
       （mysql/redis/minio 的 host 会被 compose 环境变量覆盖，可留占位）

  2) 在阿里云控制台「安全组」放行端口：80、9527、9000、9001
     （3306/6379 建议不要对公网开放）

  3) 首次启动（会拉取 gemma3:270m 模型，稍慢）：
       cd $DEPLOY_DIR && docker compose -f docker-compose.prod.yml up -d

其余的更新交给 GitHub Actions，不要再手动操作。
EOF
