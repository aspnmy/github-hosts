#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 切换到脚本所在目录（即 scripts/）
cd "$(dirname "$0")"

# 检查 go.mod 文件
if [ ! -f "go.mod" ]; then
    echo -e "${YELLOW}初始化 Go 模块...${NC}"
    go mod init github.com/aspnmy/github-hosts/scripts
    go mod tidy
fi

# 从项目根目录的 .version 文件读取版本号
VERSION_FILE="../.version"
if [ -f "$VERSION_FILE" ]; then
    VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
    if [ -z "$VERSION" ]; then
        VERSION="v0.0.0.1_nokv"
    fi
    echo -e "${YELLOW}使用 .version 文件中的版本号: ${VERSION}${NC}"
else
    VERSION="v0.0.0.1_nokv"
    echo -e "${YELLOW}未找到 .version 文件，使用默认版本号: ${VERSION}${NC}"
fi

# 构建目录和二进制名（与 workflow 保持一致）
BUILD_DIR="../build"
BINARY_NAME="github-hosts"

# 支持的平台（与 workflow 保持一致：7 个平台）
PLATFORMS=(
    "windows/amd64"
    "windows/386"
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/386"
    "linux/arm64"
)

# 清理并创建构建目录
echo -e "${YELLOW}清理构建目录...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# 遍历平台进行构建
for PLATFORM in "${PLATFORMS[@]}"; do
    # 分割平台信息
    IFS='/' read -r GOOS GOARCH <<< "$PLATFORM"

    # 构建输出文件名（与 workflow 保持一致：name.{os}-{arch}[.exe]）
    OUTPUT="${BUILD_DIR}/${BINARY_NAME}.${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    echo -e "${YELLOW}正在构建 $GOOS/$GOARCH -> ${OUTPUT}...${NC}"

    # 执行构建：注入版本号到 main.Version
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.Version=${VERSION}" -o "$OUTPUT" .

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 构建成功: ${OUTPUT}${NC}"
    else
        echo -e "${RED}✗ 构建失败: $GOOS/$GOARCH${NC}"
    fi
done

# 复制通用 Shell 脚本到构建目录（与 workflow 保持一致）
if [ -f "github_hosts.sh" ]; then
    cp github_hosts.sh "$BUILD_DIR/"
    echo -e "${GREEN}✓ 已复制 github_hosts.sh 到 ${BUILD_DIR}/github_hosts.sh${NC}"
else
    echo -e "${YELLOW}警告: 未找到 github_hosts.sh，跳过复制${NC}"
fi

# 创建 zip 压缩包（与 workflow 保持一致）
echo -e "${YELLOW}创建 zip 压缩包并清理原始文件...${NC}"
cd "$BUILD_DIR"
for FILE in *; do
    if [ -f "$FILE" ]; then
        zip -j "${FILE}.zip" "$FILE" > /dev/null
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ 已创建: ${FILE}.zip${NC}"
        else
            echo -e "${RED}✗ 压缩失败: ${FILE}${NC}"
        fi
    fi
done

# 删除非 zip 文件（只保留压缩包，与 workflow 逻辑一致）
find . -maxdepth 1 -type f ! -name "*.zip" -delete

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  构建完成！版本: ${VERSION}${NC}"
echo -e "${GREEN}  构建目录: ${BUILD_DIR}${NC}"
echo -e "${GREEN}========================================${NC}"
