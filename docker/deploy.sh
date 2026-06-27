#!/bin/bash
# Deploy GitHub Hosts Resolver to jump server
# Usage: ./deploy.sh [port]
set -e
JUMP_HOST="192.227.145.134"
JUMP_PORT="622"
SSH_KEY="/home/aspnmy/mybuild/.aspnmy/dev_worker"
PORT="${1:-18763}"

echo "=== Deploy (port: $PORT) ==="

# Copy files
ssh -i "$SSH_KEY" -p "$JUMP_PORT" -o StrictHostKeyChecking=no "$JUMP_USER@$JUMP_HOST" "mkdir -p /aspnmy/my_build/github-hosts-docker" 2>/dev/null
scp -i "$SSH_KEY" -P "$JUMP_PORT" -o StrictHostKeyChecking=no \
  Dockerfile entrypoint.sh root@192.227.145.134:/aspnmy/my_build/github-hosts-docker/

# Build & run
ssh -i "$SSH_KEY" -p "$JUMP_PORT" -o StrictHostKeyChecking=no root@192.227.145.134 << CMDS
  cd /aspnmy/my_build/github-hosts-docker
  docker stop github-hosts-resolver 2>/dev/null || true
  docker rm github-hosts-resolver 2>/dev/null || true
  docker build -t github-hosts-resolver:latest .
  docker run -d --name github-hosts-resolver --restart unless-stopped \
    -p $PORT:$PORT \
    -e PORT=$PORT -e INTERVAL=28800 \
    -v /data/github-hosts:/data \
    github-hosts-resolver:latest
  echo "Container: $(docker ps --filter name=github-hosts-resolver --format '{{.Names}} {{.Status}}')"
CMDS

# Local tunnel: localhost:$PORT -> jump:localhost:$PORT
kill \$(lsof -ti :$PORT 2>/dev/null) 2>/dev/null || true
ssh -o StrictHostKeyChecking=no -o ServerAliveInterval=30 -fN -L $PORT:localhost:$PORT \
  -p $JUMP_PORT -i "$SSH_KEY" root@$JUMP_HOST 2>/dev/null

echo "Done: http://127.0.0.1:$PORT/hosts.txt"
