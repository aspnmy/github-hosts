#!/bin/bash
# 部署 GitHub Hosts Resolver 到跳板机
set -e

JUMP_HOST="192.227.145.134"
JUMP_PORT="622"
JUMP_USER="root"
SSH_KEY="/home/aspnmy/mybuild/.aspnmy/dev_worker"
LOCAL_DIR="/home/aspnmy/mybuild/github-hosts/docker"
REMOTE_DIR="/aspnmy/my_build/github-hosts-docker"

echo "=== 部署 GitHub Hosts Resolver ==="

# 1. 复制文件到跳板机
echo "复制文件到跳板机..."
ssh -i "$SSH_KEY" -p "$JUMP_PORT" -o StrictHostKeyChecking=no "$JUMP_USER@$JUMP_HOST" "mkdir -p $REMOTE_DIR"
scp -i "$SSH_KEY" -P "$JUMP_PORT" -o StrictHostKeyChecking=no \
  "$LOCAL_DIR/Dockerfile" "$LOCAL_DIR/entrypoint.sh" \
  "$JUMP_USER@$JUMP_HOST:$REMOTE_DIR/"

# 2. 构建并运行容器
echo "构建并启动容器..."
ssh -i "$SSH_KEY" -p "$JUMP_PORT" -o StrictHostKeyChecking=no "$JUMP_USER@$JUMP_HOST" << 'CMDS'
  cd /aspnmy/my_build/github-hosts-docker

  # 停止旧容器
  docker stop github-hosts-resolver 2>/dev/null || true
  docker rm github-hosts-resolver 2>/dev/null || true

  # 构建新镜像
  docker build -t github-hosts-resolver:latest .

  # 启动容器（host网络模式方便访问）
  docker run -d \
    --name github-hosts-resolver \
    --restart unless-stopped \
    -p 5000:5000 \
    -v /data/github-hosts:/data \
    -e PORT=5000 \
    -e INTERVAL=28800 \
    github-hosts-resolver:latest

  echo ""
  echo "=== 容器状态 ==="
  docker ps --filter name=github-hosts-resolver --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
  echo ""
  echo "=== 启动日志 ==="
  sleep 5
  docker logs github-hosts-resolver --tail 20
CMDS

echo ""
echo "=== 部署完成 ==="
echo "访问: http://$JUMP_HOST:5000/hosts.txt"
echo "本地隧道: curl http://127.0.0.1:5000/hosts.txt"
