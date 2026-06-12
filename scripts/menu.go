package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// showHostsContent 显示 hosts 文件内容
func (app *App) showHostsContent() error {
	// 读取 hosts 文件内容
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("读取 hosts 文件失败: %w", err)
	}

	// 显示完整内容
	fmt.Printf("\n当前 hosts 文件内容 (%s)：\n", hostsFile)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(string(content))
	fmt.Println(strings.Repeat("-", 80))

	// 显示文件信息
	if info, err := os.Stat(hostsFile); err == nil {
		fmt.Printf("文件大小: %.2f KB\n", float64(info.Size())/1024)
		fmt.Printf("修改时间: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	}

	return nil
}

// openConfigDir 打开配置目录
func (app *App) openConfigDir() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", app.baseDir)
	case "linux":
		cmd = exec.Command("xdg-open", app.baseDir)
	case "windows":
		cmd = exec.Command("explorer", app.baseDir)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("打开目录失败: %w", err)
	}

	return nil
}

// openConfigFile 直接打开配置文件 config.json
//
// 参数：
//   - 无
//
// 返回值：
//   - error: 如果系统命令执行失败或路径无效时返回错误
//
// 说明：
//   - Windows: 使用 notepad 记事本打开 config.json
//   - macOS:   使用 open 命令（默认用文本编辑器打开）
//   - Linux:   使用 xdg-open（默认关联的文本编辑器打开）
//   - 执行前会检查 config.json 文件是否存在，不存在则给出提示
func (app *App) openConfigFile() error {
	// 检查配置文件是否存在
	if _, err := os.Stat(app.configFile); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s（请先执行安装操作）", app.configFile)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS 使用 open -e 强制用默认文本编辑器打开
		cmd = exec.Command("open", "-e", app.configFile)
	case "linux":
		// Linux 使用 xdg-open
		cmd = exec.Command("xdg-open", app.configFile)
	case "windows":
		// Windows 使用 notepad 打开
		cmd = exec.Command("notepad", app.configFile)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("打开配置文件失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "已打开配置文件: %s", app.configFile)
	return nil
}

// showUpdateLogs 显示更新日志
func (app *App) showUpdateLogs() error {
	// 使用配置的时区生成日志文件名
	loc, _ := app.getConfigTimeZone()
	now := time.Now().In(loc)
	logFile := filepath.Join(app.logDir, fmt.Sprintf("update_%s.log", now.Format("20060102")))

	content, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			app.logWithLevel(INFO, "今日暂无更新日志")
			return nil
		}
		return fmt.Errorf("读取日志文件失败: %w", err)
	}

	fmt.Println("\n最近的更新日志:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(string(content))
	fmt.Println(strings.Repeat("-", 80))
	return nil
}

// checkStatus 检查系统状态
func (app *App) checkStatus() error {
	app.logWithLevel(INFO, "开始检查系统状态...")

	// 1. 检查配置文件
	config, err := app.loadConfig()
	if err != nil {
		app.logWithLevel(ERROR, "配置文件检查失败: %v", err)
	} else {
		// 使用配置的时区格式化时间
		loc, tzName := app.getConfigTimeZone()
		lastUpdateStr := "(未记录)"
		if !config.LastUpdate.IsZero() {
			lastUpdateStr = config.LastUpdate.In(loc).Format("2006-01-02 15:04:05 MST")
		}

		app.logWithLevel(INFO, "配置文件状态:")
		app.logWithLevel(INFO, "  • 更新间隔: %d 分钟", config.UpdateInterval)
		app.logWithLevel(INFO, "  • 自动更新: %s", map[bool]string{true: "已启用", false: "已禁用"}[config.AutoUpdate])
		app.logWithLevel(INFO, "  • 系统时区: %s", tzName)
		app.logWithLevel(INFO, "  • 最后更新: %s", lastUpdateStr)
		app.logWithLevel(INFO, "  • 版本: %s", config.Version)
	}

	// 2. 检查 hosts 文件
	if _, err := os.Stat(hostsFile); err != nil {
		app.logWithLevel(ERROR, "hosts 文件检查失败: %v", err)
	} else {
		content, err := os.ReadFile(hostsFile)
		if err != nil {
			app.logWithLevel(ERROR, "读取 hosts 文件失败: %v", err)
		} else {
			lines := strings.Split(string(content), "\n")
			githubCount := 0
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "github") {
					githubCount++
				}
			}
			app.logWithLevel(INFO, "hosts 文件状态:")
			app.logWithLevel(INFO, "  • 文件大小: %.2f KB", float64(len(content))/1024)
			app.logWithLevel(INFO, "  • GitHub 相关记录数: %d", githubCount)
		}
	}

	// 3. 检查定时任务状态
	app.logWithLevel(INFO, "定时任务状态:")
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("launchctl", "list", "com.github.hosts")
		if err := cmd.Run(); err == nil {
			app.logWithLevel(SUCCESS, "  • 定时任务运行正常")
		} else {
			app.logWithLevel(WARNING, "  • 定时任务未运行")
		}
	case "windows":
		cmd := exec.Command("schtasks", "/query", "/tn", windowsTaskName)
		if err := cmd.Run(); err == nil {
			app.logWithLevel(SUCCESS, "  • 定时任务运行正常")
		} else {
			app.logWithLevel(WARNING, "  • 定时任务未运行")
		}
	case "linux":
		if _, err := os.Stat(linuxCronPath); err == nil {
			app.logWithLevel(SUCCESS, "  • 定时任务配置正常")
		} else {
			app.logWithLevel(WARNING, "  • 定时任务配置不存在")
		}
	}

	// 4. 检查目录权限
	app.logWithLevel(INFO, "目录权限检查:")
	dirs := []string{app.baseDir, app.backupDir, app.logDir}
	for _, dir := range dirs {
		if err := app.checkDirPermissions(dir); err != nil {
			app.logWithLevel(WARNING, "  • %s: %v", dir, err)
		} else {
			app.logWithLevel(SUCCESS, "  • %s: 权限正常", dir)
		}
	}

	// 5. 检查备份状态
	if backups, err := app.listBackups(); err != nil {
		app.logWithLevel(ERROR, "备份检查失败: %v", err)
	} else {
		app.logWithLevel(INFO, "备份状态:")
		app.logWithLevel(INFO, "  • 备份文件数量: %d", len(backups))
		if len(backups) > 0 {
			app.logWithLevel(INFO, "  • 最新备份: %s", backups[len(backups)-1])
		}
	}

	return nil
}
