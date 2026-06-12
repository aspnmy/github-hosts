package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// 当前程序版本号由 .version 文件控制，通过 getAppVersion() 动态读取，
// 托底值为 banner.go 中定义的 defaultVersion ("v0.0.0.1_nokv")

// GitHub 仓库信息
const (
	repoOwner = "aspnmy"
	repoName  = "github-hosts"
)

// GitHubRelease GitHub Release API 返回结构（仅保留所需字段）
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []ReleaseAsset `json:"assets"`
	HtmlURL string `json:"html_url"`
}

// ReleaseAsset Release 资产（可下载文件）信息
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateInfo 版本更新信息
type UpdateInfo struct {
	NewVersion  string        // 新版本号
	DownloadURL string        // 新程序下载地址
	ReleaseURL  string        // Release 页面 URL
	AssetSize   int64         // 文件大小（字节）
}

// checkForUpdates 查询 GitHub 最新 Release，若存在更新则返回更新信息，否则返回 nil
//
// 参数：
//   - 无
//
// 返回值：
//   - *UpdateInfo: 新版本信息；若当前已是最新则返回 nil
//   - error: 查询或解析失败时返回错误
func checkForUpdates() (*UpdateInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "github-hosts-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询更新失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析 Release JSON 失败: %w", err)
	}

	// 比较版本号，若线上版本 <= 当前版本则无更新
	// 注意：release.TagName 通常格式为 "v1.0.0"，getAppVersion() 返回 "v0.0.0.1_nokv"
	// compareVersions 内部已统一处理 v 前缀与非纯数字段
	if compareVersions(release.TagName, getAppVersion()) <= 0 {
		return nil, nil
	}

	// 根据当前操作系统和架构匹配合适的下载资产
	asset := matchReleaseAsset(release.Assets)
	if asset == nil {
		// 未找到对应平台的资产，返回 Release 页面让用户手动下载
		return &UpdateInfo{
			NewVersion:  release.TagName,
			DownloadURL: "",
			ReleaseURL:  release.HtmlURL,
			AssetSize:   0,
		}, nil
	}

	return &UpdateInfo{
		NewVersion:  release.TagName,
		DownloadURL: asset.BrowserDownloadURL,
		ReleaseURL:  release.HtmlURL,
		AssetSize:   asset.Size,
	}, nil
}

// matchReleaseAsset 在 Release 资产列表中根据当前 OS/架构匹配合适的下载文件
//
// 参数：
//   - assets: Release 资产列表
//
// 返回值：
//   - *ReleaseAsset: 匹配到的资产；未找到则返回 nil
func matchReleaseAsset(assets []ReleaseAsset) *ReleaseAsset {
	osExt := ""
	switch runtime.GOOS {
	case "windows":
		osExt = ".exe"
	}

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		// 优先匹配：名称中同时包含当前 OS 和当前架构
		if strings.Contains(name, runtime.GOOS) &&
		   strings.Contains(name, runtime.GOARCH) &&
		   (osExt == "" || strings.HasSuffix(name, osExt)) {
			return &asset
		}
	}

	// 回退：仅匹配操作系统 + 可执行后缀
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, runtime.GOOS) &&
		   (osExt == "" || strings.HasSuffix(name, osExt)) {
			return &asset
		}
	}

	// 再回退：仅匹配可执行后缀（适用于只打了一个平台的 Release）
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if osExt == "" || strings.HasSuffix(name, osExt) {
			return &asset
		}
	}

	return nil
}

// compareVersions 比较两个语义化版本号（支持 v 前缀，自动处理）
//
// 参数：
//   - v1: 版本号 1
//   - v2: 版本号 2
//
// 返回值：
//   -  1: v1 > v2
//   -  0: v1 == v2
//   - -1: v1 < v2
func compareVersions(v1, v2 string) int {
	// 去掉 v/V 前缀
	v1 = strings.TrimPrefix(strings.TrimPrefix(v1, "v"), "V")
	v2 = strings.TrimPrefix(strings.TrimPrefix(v2, "v"), "V")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// 取最长长度，缺失段以 0 补齐
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			num1 = parseNumericSegment(parts1[i])
		}
		if i < len(parts2) {
			num2 = parseNumericSegment(parts2[i])
		}
		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}
	return 0
}

// parseNumericSegment 从版本段字符串中提取前置数字部分
//
// 参数：
//   - s: 版本段字符串，例如 "10"、"1_nokv"、"3beta"
//
// 返回值：
//   - int: 提取出的数字；无法解析时返回 0
//
// 示例：
//   parseNumericSegment("10")     → 10
//   parseNumericSegment("1_nokv") → 1
//   parseNumericSegment("")       → 0
func parseNumericSegment(s string) int {
	// 先尝试直接整段转换
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	// 整段非纯数字，提取前缀数字部分
	var numStr string
	for _, r := range s {
		if r >= '0' && r <= '9' {
			numStr += string(r)
		} else {
			break
		}
	}
	if numStr == "" {
		return 0
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

// getDownloadDir 获取下载临时目录（优先 /tmp/downloads，Windows 下使用 %TEMP%/downloads）
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 临时下载目录的绝对路径
func getDownloadDir() string {
	tmpDir := os.TempDir()
	downloadDir := filepath.Join(tmpDir, "downloads")
	return downloadDir
}

// downloadFile 从指定 URL 下载文件到下载目录，返回保存的本地路径
//
// 参数：
//   - url: 下载地址
//   - filename: 本地保存的文件名（不含目录）
//
// 返回值：
//   - string: 保存的本地文件绝对路径
//   - error: 下载失败时返回错误
func downloadFile(url, filename string) (string, error) {
	downloadDir := getDownloadDir()
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("创建下载目录失败: %w", err)
	}

	savePath := filepath.Join(downloadDir, filename)

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "github-hosts-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载返回状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建下载文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(savePath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 下载成功后赋予可执行权限（Unix 平台）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(savePath, 0755); err != nil {
			return savePath, err // 警告级别，不阻塞
		}
	}

	return savePath, nil
}

// getCurrentExecutablePath 获取当前可执行文件的绝对路径
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 当前可执行文件的绝对路径
//   - error: 获取失败时返回错误
func getCurrentExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取当前执行路径失败: %w", err)
	}
	// 解析符号链接，得到真实路径
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil // 无法解析 symlink 时退化为原始路径
	}
	return real, nil
}

// performUpdate 执行版本更新：将当前程序备份，把新版本移到原位置
//
// 参数：
//   - newExePath: 已下载的新版本可执行文件路径
//   - newVersion: 新版本号，用于生成备份文件名
//
// 返回值：
//   - error: 更新失败时返回错误；成功时不会返回（程序将退出）
//
// 工作流程：
//   1. 在当前程序目录生成平台相关的更新脚本（Windows .bat，Unix .sh）
//   2. 脚本内容：等待当前进程退出 -> 备份旧程序 -> 将新版本移动到原位置 -> 可选重启
//   3. 启动脚本并立即退出当前程序
func performUpdate(newExePath, newVersion string) error {
	currentExe, err := getCurrentExecutablePath()
	if err != nil {
		return err
	}

	workDir := filepath.Dir(currentExe)
	exeName := filepath.Base(currentExe)
	backupName := fmt.Sprintf("%s_%s", exeName, getAppVersion())
	backupPath := filepath.Join(workDir, backupName)

	var scriptPath string
	var scriptContent string

	switch runtime.GOOS {
	case "windows":
		// Windows 批处理脚本
		scriptPath = filepath.Join(workDir, "_update.bat")
		scriptContent = fmt.Sprintf(`@echo off
echo Waiting for program to exit...
timeout /t 2 /nobreak >nul
echo Backing up current version...
move /Y "%s" "%s"
echo Installing new version...
move /Y "%s" "%s"
echo Update completed. New version: %s
del "%%~f0"
`, currentExe, backupPath, newExePath, currentExe, newVersion)

	default:
		// Linux / macOS Shell 脚本
		scriptPath = filepath.Join(workDir, "_update.sh")
		scriptContent = fmt.Sprintf(`#!/bin/sh
sleep 2
echo "Backing up current version..."
mv -f "%s" "%s"
echo "Installing new version..."
mv -f "%s" "%s"
chmod +x "%s"
echo "Update completed. New version: %s"
rm -f "$0"
`, currentExe, backupPath, newExePath, currentExe, currentExe, newVersion)
	}

	// 写入脚本
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("写入更新脚本失败: %w", err)
	}

	// 启动更新脚本，然后退出当前程序
	fmt.Println("")
	fmt.Printf("  新版本已下载到: %s\n", newExePath)
	fmt.Printf("  当前程序将备份为: %s\n", backupName)
	fmt.Println("  程序即将退出并执行更新，请稍候...")
	fmt.Println("")

	if runtime.GOOS == "windows" {
		// Windows 下用 cmd /c start /B 在后台运行批处理
		cmd := exec.Command("cmd", "/c", "start", "/B", scriptPath)
		cmd.Dir = workDir
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动更新脚本失败: %w", err)
		}
	} else {
		// Unix 下用 sh 在后台运行脚本，输出/错误均丢弃（不阻塞当前进程）
		cmd := exec.Command("sh", scriptPath)
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动更新脚本失败: %w", err)
		}
		_ = cmd.Process.Release() // 与子进程解耦
	}

	// 立即退出当前程序
	os.Exit(0)
	return nil // 不会执行到这里
}

// runUpdateCheck 执行「检查更新 → 询问用户 → 下载 → 更新」完整流程
//
// 参数：
//   - app: App 实例（用于日志输出），可为 nil
//
// 返回值：
//   - error: 流程中任意步骤失败时返回错误
func runUpdateCheck(app *App) error {
	fmt.Println("")
	fmt.Println("====== 检查程序更新 ======")
	fmt.Printf("  当前版本: %s\n", getAppVersion())
	fmt.Println("  正在查询 GitHub Releases...")

	info, err := checkForUpdates()
	if err != nil {
		if app != nil {
			app.logWithLevel(WARNING, fmt.Sprintf("查询更新失败: %v", err))
		} else {
			fmt.Printf("  [警告] 查询更新失败: %v\n", err)
		}
		return nil
	}

	if info == nil {
		fmt.Println("  ✅ 当前已是最新版本，无需更新")
		return nil
	}

	// 发现新版本
	fmt.Println("")
	fmt.Printf("  🎉 发现新版本: %s (当前: %s)\n", info.NewVersion, getAppVersion())
	if info.ReleaseURL != "" {
		fmt.Printf("  Release 页面: %s\n", info.ReleaseURL)
	}

	// 若未找到对应平台的下载资产，只提示不自动下载
	if info.DownloadURL == "" {
		fmt.Println("")
		fmt.Println("  未找到适配当前平台的自动下载资产，请手动访问 Release 页面下载。")
		return nil
	}

	if info.AssetSize > 0 {
		fmt.Printf("  文件大小: %.2f MB\n", float64(info.AssetSize)/1024/1024)
	}
	fmt.Printf("  下载地址: %s\n", info.DownloadURL)

	// 询问用户是否更新
	fmt.Print("\n是否下载并更新到新版本？[Y/n]: ")
	var choice string
	fmt.Scanf("%s", &choice)

	if choice == "n" || choice == "N" {
		fmt.Println("  已取消更新")
		return nil
	}

	// 下载
	fmt.Println("")
	fmt.Println("  正在下载新版本...")
	filename := filepath.Base(info.DownloadURL)
	if filename == "" || strings.HasPrefix(filename, "/") {
		filename = fmt.Sprintf("github-hosts-%s", info.NewVersion)
		if runtime.GOOS == "windows" {
			filename += ".exe"
		}
	}

	savedPath, err := downloadFile(info.DownloadURL, filename)
	if err != nil {
		return fmt.Errorf("下载新版本失败: %w", err)
	}
	fmt.Printf("  ✅ 下载完成: %s\n", savedPath)

	// 执行更新（此函数内部会调用 os.Exit 结束当前程序）
	return performUpdate(savedPath, info.NewVersion)
}

// runSilentUpdateCheck 在程序启动时静默检查更新，若发现新版本给出提示（不阻塞、不自动下载）
//
// 参数：
//   - 无
//
// 返回值：
//   - 无（所有内部错误吞掉，仅输出友好信息）
func runSilentUpdateCheck() {
	info, err := checkForUpdates()
	if err != nil {
		// 静默模式下不打印错误，避免启动时打扰用户
		return
	}
	if info != nil {
		fmt.Println("")
		fmt.Printf("  🎉 发现新版本 %s，当前版本 %s\n", info.NewVersion, getAppVersion())
		fmt.Printf("  可在主菜单中选择「检查更新」来升级，或访问: %s\n", info.ReleaseURL)
		fmt.Println("")
	}
}
