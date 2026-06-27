#!/bin/sh
set -e

DOMAINS_URL="${DOMAINS_URL:-https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt}"
INTERVAL="${INTERVAL:-28800}"
PORT="${PORT:-5000}"
DNS_SERVER="${DNS_SERVER:-1.1.1.1}"

echo "=== GitHub Hosts Resolver ==="
echo "Domain file: $DOMAINS_URL"
echo "Interval: $((INTERVAL/3600))h"
echo "Port: $PORT"
echo "DNS: $DNS_SERVER"

resolve_all() {
    echo "[$(date)] Starting resolve..."
    if ! curl -sL --connect-timeout 10 --max-time 30 "$DOMAINS_URL" -o /tmp/domains.txt 2>/dev/null; then
        echo "Download failed, using cache"
        [ -f /tmp/domains_cache.txt ] && cp /tmp/domains_cache.txt /tmp/domains.txt || return 1
    else
        cp /tmp/domains.txt /tmp/domains_cache.txt
    fi

    total=$(wc -l < /tmp/domains.txt)
    count=0; ok=0; fail=0

    { echo "# GitHub Hosts - Auto Generated"
      echo "# Updated: $(TZ=Asia/Shanghai date '+%Y-%m-%d %H:%M:%S')"
      echo "# Total domains: $total"
      echo "# Source: https://github.com/aspnmy/github-hosts"
      echo ""
    } > /tmp/hosts.txt

    while IFS= read -r domain; do
        [ -z "$domain" ] && continue
        ip=$(dig +short @$DNS_SERVER "$domain" 2>/dev/null | grep -E '^[0-9]+.[0-9]+.[0-9]+.[0-9]+$' | grep -v '^127.' | head -1)
        if [ -n "$ip" ]; then
            printf "%-25s %s\n" "$ip" "$domain" >> /tmp/hosts.txt
            ok=$((ok + 1))
        else
            fail=$((fail + 1))
        fi
        count=$((count + 1))
    done < /tmp/domains.txt

    { echo ""; echo "# OK: $ok | Failed: $fail | Total: $total"; } >> /tmp/hosts.txt
    cp /tmp/hosts.txt /data/hosts.txt
    echo "{\"total\":$total,\"success\":$ok,\"failed\":$fail,\"updated\":\"$(date -Iseconds)\"}" > /data/status.json
    echo "[$(date)] Done: $ok OK, $fail failed"
}

resolve_all

# HTTP server
mkdir -p /data/www
ln -sf /data/hosts.txt /data/www/hosts.txt
ln -sf /data/status.json /data/www/status.json

INDEX='<!DOCTYPE html><html><head><meta charset="utf-8"><title>GitHub Hosts</title></head><body><h1>GitHub Hosts</h1><p><a href="hosts.txt">hosts.txt</a></p><p><a href="status.json">status.json</a></p><pre id="s">Loading...</pre><script>fetch("status.json").then(r=>r.json()).then(d=>{document.getElementById("s").textContent="Updated: "+d.updated+"\nOK: "+d.success+" Failed: "+d.failed+" Total: "+d.total})</script></body></html>'
echo "$INDEX" > /data/www/index.html

echo "=== HTTP Server on port $PORT ==="
cd /data/www && busybox httpd -f -p "$PORT" -h /data/www
