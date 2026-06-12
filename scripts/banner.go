package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultVersion 版本号托底值：当 .version 文件不存在或为空时使用
const defaultVersion = "v0.0.0.1_nokv"

// Version 编译时通过 ldflags 注入的版本号
// 编译时使用：go build -ldflags "-X main.Version=v0.0.0.2_nokv"
var Version string

// getVersionFilePath 获取 .version 文件的绝对路径（位于当前可执行文件所在目录）
//
// 参数：
//   - 无
//
// 返回值：
//   - string: .version 文件的绝对路径
//   - error: 无法获取执行路径时返回错误
func getVersionFilePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe
	}
	return filepath.Join(filepath.Dir(real), ".version"), nil
}

// getAppVersion 获取当前程序的版本号
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 版本号；优先使用编译时注入的版本，其次读取 .version 文件，最后使用托底值
//
// 优先级：
//  1. 编译时通过 -ldflags "-X main.Version=xxx" 注入的版本号
//  2. 可执行文件同目录下的 .version 文件
//  3. 托底值 defaultVersion
func getAppVersion() string {
	// 优先级 1: ldflags 注入的版本号
	if Version != "" {
		return Version
	}

	// 优先级 2: .version 文件
	versionPath, err := getVersionFilePath()
	if err == nil {
		content, err := os.ReadFile(versionPath)
		if err == nil {
			version := strings.TrimSpace(string(content))
			if version != "" {
				return version
			}
		}
	}

	// 优先级 3: 托底值
	return defaultVersion
}

// ensureVersionFile 确保 .version 文件存在；若不存在则写入当前版本号
//
// 参数：
//   - 无
//
// 返回值：
//   - error: 写入失败时返回错误
//
// 说明：
//  1. 调用 getAppVersion() 得到当前应使用的版本号
//  2. 若 .version 文件不存在或为空，写入该版本号
//  3. 若文件已存在且有内容，保持原样不动
func ensureVersionFile() error {
	versionPath, err := getVersionFilePath()
	if err != nil {
		return fmt.Errorf("无法确定 .version 文件路径: %w", err)
	}

	// 检查文件是否存在且有有效内容
	content, err := os.ReadFile(versionPath)
	if err == nil && strings.TrimSpace(string(content)) != "" {
		return nil // 文件已存在且有内容，不做修改
	}

	// 文件不存在或为空，写入当前版本
	current := getAppVersion()
	if err := os.WriteFile(versionPath, []byte(current+"\n"), 0644); err != nil {
		return fmt.Errorf("写入 .version 文件失败: %w", err)
	}
	return nil
}

// getBanner 生成带当前版本号的程序 Banner 文本
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 完整的 Banner 字符串（含末尾换行）
func getBanner() string {
	return fmt.Sprintf(`   _______ __  __          __    __          __  __
  / ____(_) /_/ /_  __  __/ /_  / /_  ____  / /_/ /______
 / / __/ / __/ __ \/ / / / __ \/ __ \/ __ \/ __/ __/ ___/
/ /_/ / / /_/ / / / /_/ / / / / /_/ / /_/ / /_/ /_(__  )
\____/_/\__/_/ /_/\__,_/_/ /_/\____/\____/\__/\__/_____/
GitHub Hosts noKv Manager - https://github.com/aspnmy/github-hosts
Version: %s
加速访问 GitHub
`, getAppVersion())
}
