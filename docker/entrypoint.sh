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
    echo "[$(date)] Resolving $total domains..."
    if ! curl -sL --connect-timeout 10 --max-time 30 "$DOMAINS_URL" -o /tmp/domains.txt 2>/dev/null; then
        [ -f /tmp/domains_cache.txt ] && cp /tmp/domains_cache.txt /tmp/domains.txt || return 1
    else
        cp /tmp/domains.txt /tmp/domains_cache.txt
    fi
    total=$(wc -l < /tmp/domains.txt)
    count=0; ok=0; fail=0
    {
        echo "# GitHub Hosts - Auto Generated"
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

# First resolve
resolve_all

# Start Python HTTP server in background
python3 -m http.server -d /data $PORT &
HTTP_PID=$!
echo "HTTP server started (PID: $HTTP_PID, port: $PORT)"

# Loop: re-resolve every INTERVAL seconds
while true; do
    sleep $INTERVAL
    resolve_all
done
