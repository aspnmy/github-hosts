#!/bin/sh
set -e

DOMAINS_URL="${DOMAINS_URL:-https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt}"
INTERVAL="${INTERVAL:-28800}"  # 8小时
PORT="${PORT:-5000}"
DNS_SERVER="${DNS_SERVER:-1.1.1.1}"

echo "=== GitHub Hosts Resolver ==="
echo "域名文件: $DOMAINS_URL"
echo "更新间隔: $((INTERVAL/3600))小时"
echo "监听端口: $PORT"
echo "DNS服务器: $DNS_SERVER"
echo ""

resolve_all() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 开始解析..."

    # 下载域名列表
    if ! curl -sL --connect-timeout 10 --max-time 30 "$DOMAINS_URL" -o /tmp/domains.txt; then
        echo "  ! 下载失败，使用缓存"
        [ -f /tmp/domains_cache.txt ] && cp /tmp/domains_cache.txt /tmp/domains.txt || return 1
    else
        cp /tmp/domains.txt /tmp/domains_cache.txt
    fi

    total=$(wc -l < /tmp/domains.txt)
    count=0
    success=0
    failed=0

    # 生成 hosts 文件头
    {
        echo "# GitHub Hosts - 自动解析"
        echo "# 更新: $(TZ='Asia/Shanghai' date '+%Y-%m-%d %H:%M:%S')"
        echo "# 域名总数: $total"
        echo ""
    } > /tmp/hosts.txt

    # 批量 DNS 解析
    while IFS= read -r domain; do
        [ -z "$domain" ] && continue
        ip=$(dig +short @"$DNS_SERVER" "$domain" 2>/dev/null | \
             grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | \
             grep -v '^127\.' | head -1)
        if [ -n "$ip" ]; then
            printf "%-25s %s\n" "$ip" "$domain" >> /tmp/hosts.txt
            success=$((success + 1))
        else
            echo "# FAILED: $domain" >> /tmp/hosts.txt
            failed=$((failed + 1))
        fi
        count=$((count + 1))
        if [ $((count % 50)) -eq 0 ]; then
            echo "  进度: $count/$total"
        fi
    done < /tmp/domains.txt

    {
        echo ""
        echo "# 成功: $success | 失败: $failed | 总计: $total"
    } >> /tmp/hosts.txt

    cp /tmp/hosts.txt /data/hosts.txt
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 完成: $success 成功, $failed 失败"
}

# 启动 HTTP 服务（后台）
echo "=== 启动 HTTP 服务 ==="
cat > /tmp/httpd.sh << 'HTTP'
#!/bin/sh
while true; do
    echo -e "HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nAccess-Control-Allow-Origin: *\r\n\r\n$(cat /data/hosts.txt 2>/dev/null)" | nc -l -p $PORT -q 1 > /dev/null 2>&1
done
HTTP
chmod +x /tmp/httpd.sh
sh /tmp/httpd.sh &
HTTP_PID=$!

# 首次立即解析
resolve_all

# 定时循环
while true; do
    sleep $INTERVAL
    resolve_all
done
