#!/bin/bash
# 这个脚本用于从远程 URL 下载 GitHub Hosts 区块，并更新本地 /etc/hosts 文件
# 主要用于特殊的阉割版系统，比如UgOS 旧版本，或者其他需要定期更新 GitHub Hosts 的系统

URL="https://hosts.earth-online.org/hosts"
BACKUP_DIR="/etc/hosts.backup"
LOCAL_HOSTS="/etc/hosts"
CURRENT_DATE=$(date +%Y%m%d_%H%M%S)
TEMP_DOWNLOAD=$(mktemp)
TEMP_BLOCK=$(mktemp)
TEMP_OUTPUT=$(mktemp)

cleanup() {
    rm -f "$TEMP_DOWNLOAD" "$TEMP_BLOCK" "$TEMP_OUTPUT"
}
trap cleanup EXIT

if [ "$EUID" -ne 0 ]; then
    echo "错误：请使用 sudo 或以 root 身份运行此脚本"
    exit 1
fi

# 1. 下载
echo "[1/5] 正在下载远程 hosts..."
if ! curl -sSL "$URL" -o "$TEMP_DOWNLOAD"; then
    echo "错误：下载失败"
    exit 1
fi

# 2. 提取区块：从 "# github hosts" 到包含 "更新时间" 的行（大小写不敏感，忽略前导空格）
echo "[2/5] 从下载内容中提取 GitHub Hosts 区块..."
awk '
tolower($0) ~ /^[[:space:]]*# github hosts/,
tolower($0) ~ /更新时间/
' "$TEMP_DOWNLOAD" > "$TEMP_BLOCK"

if [ ! -s "$TEMP_BLOCK" ]; then
    echo "错误：未找到 '# github hosts' 到 '更新时间' 的区块"
    exit 1
fi

# 3. 检查结束标志（最后一行必须包含 "更新时间"）
if ! tail -n1 "$TEMP_BLOCK" | grep -qi "更新时间"; then
    echo "错误：提取的区块最后一行不包含 '更新时间'，可能不完整"
    exit 1
fi

# 4. 备份
echo "[3/5] 备份当前 hosts 文件..."
mkdir -p "$BACKUP_DIR"
cp "$LOCAL_HOSTS" "${BACKUP_DIR}/hosts_${CURRENT_DATE}"
echo "备份到 ${BACKUP_DIR}/hosts_${CURRENT_DATE}"

# 5. 替换或追加（匹配本地开始行也使用 # github hosts，结束行使用 # 更新时间 或 # update）
if grep -qi "^[[:space:]]*# github hosts" "$LOCAL_HOSTS"; then
    echo "[4/5] 本地存在区块，正在替换..."
    # 核心替换逻辑：匹配开始行到任意结束行（# update 或 # 更新时间）
    awk '
    BEGIN { inside=0; replaced=0 }
    {
        if (tolower($0) ~ /^[[:space:]]*# github hosts/ && !replaced) {
            inside=1
            while ((getline line < "'"$TEMP_BLOCK"'") > 0) print line
            close("'"$TEMP_BLOCK"'")
            replaced=1
            next
        }
        # 结束行：无论是 # update 还是 # 数据更新时间
        if (inside && (tolower($0) ~ /^[[:space:]]*# update[[:space:]]/ || tolower($0) ~ /^[[:space:]]*# 数据更新时间/)) {
            inside=0
            next
        }
        if (!inside) print
    }
    ' "$LOCAL_HOSTS" > "$TEMP_OUTPUT"

    if cp "$TEMP_OUTPUT" "$LOCAL_HOSTS"; then
        echo "✅ 替换成功！仅更新了 GitHub 相关 Hosts"
    else
        echo "❌ 写回失败，已恢复备份"
        cp "${BACKUP_DIR}/hosts_${CURRENT_DATE}" "$LOCAL_HOSTS"
        exit 1
    fi
else
    echo "[4/5] 本地未找到区块，正在追加到文件末尾..."
    echo "" >> "$LOCAL_HOSTS"
    cat "$TEMP_BLOCK" >> "$LOCAL_HOSTS"
    echo "✅ 已追加区块"
fi

# 6. 刷新 DNS 缓存
echo "[5/5] 刷新 DNS 缓存..."
command -v systemd-resolve &> /dev/null && systemd-resolve --flush-caches 2>/dev/null || true
command -v resolvectl &> /dev/null && resolvectl flush-caches 2>/dev/null || true
command -v service &> /dev/null && service nscd reload 2>/dev/null || true

echo ""
echo "========== 更新完成 =========="
echo "其他自定义 Hosts 条目均被保留"
