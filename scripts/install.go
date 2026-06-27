package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (app *App) installMenu() error {
	app.logWithLevel(INFO, "检查系统状态...")
	content, err := os.ReadFile(hostsFile)
	if err == nil && strings.Contains(string(content), "GitHub Hosts") {
		fmt.Print("\n检测到已存在 GitHub Hosts 数据，是否要更新？[Y/n]: ")
		var r string
		fmt.Scanf("%s", &r)
		if r == "n" || r == "N" {
			app.logWithLevel(INFO, "已取消更新操作")
			return nil
		}
		if err := app.updateHosts(); err != nil {
			return fmt.Errorf("更新 hosts 失败: %w", err)
		}
		app.logWithLevel(SUCCESS, "hosts 文件更新完成")
		return nil
	}

	app.logWithLevel(INFO, "开始安装配置向导...")
	var autoUpdate bool = true
	fmt.Print("\n是否开启自动更新？[Y/n]: ")
	var r string
	fmt.Scanf("%s", &r)
	var interval int = 60
	if r == "n" || r == "N" {
		autoUpdate = false
	} else {
		fmt.Println("\n请选择更新间隔：")
		fmt.Println("1. 每 30 分钟")
		fmt.Println("2. 每 60 分钟")
		fmt.Println("3. 每 120 分钟")
		fmt.Print("请输入选项 (1-3): ")
		var c int
		fmt.Scanf("%d", &c)
		switch c {
		case 1: interval = 30
		case 2: interval = 60
		case 3: interval = 120
		default: interval = 60
		}
	}

	if err := app.setupDirectories(); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := app.updateConfig(interval, autoUpdate); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}
	if err := app.updateHosts(); err != nil {
		return fmt.Errorf("更新 hosts 失败: %w", err)
	}
	if autoUpdate {
		if err := app.setupCron(interval); err != nil {
			return fmt.Errorf("设置定时任务失败: %w", err)
		}
	}

	app.logWithLevel(SUCCESS, "安装完成！")
	app.testConnection()
	return nil
}

func (app *App) setupDirectories() error {
	for _, dir := range []string{app.baseDir, app.backupDir, app.logDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (app *App) updateConfig(interval int, autoUpdate bool) error {
	config := Config{
		UpdateInterval: interval,
		LastUpdate:     time.Now().UTC(),
		Version:        "1.0.0",
		AutoUpdate:     autoUpdate,
	}
	data, _ := json.MarshalIndent(config, "", "    ")
	return os.WriteFile(app.configFile, data, 0644)
}

func (app *App) updateHosts() error {
	// 从配置文件读取数据源URL
	apiURL := loadHostSource()

	app.logWithLevel(INFO, "备份 hosts 文件")
	if err := app.backupHosts(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	app.logWithLevel(INFO, "清理旧的 GitHub Hosts 内容")
	app.cleanHostsFile()

	app.logWithLevel(INFO, "从服务器获取最新 hosts 数据")
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回状态码: %d", resp.StatusCode)
	}

	content, _ := io.ReadAll(resp.Body)
	app.logWithLevel(SUCCESS, "获取数据成功 (%d bytes)", len(content))

	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开 hosts 文件失败: %w", err)
	}
	defer f.Close()

	startMarker := fmt.Sprintf("\n# ===== GitHub Hosts Start ===== \n# (Updated: %s)\n", time.Now().Format("2006-01-02 15:04:05"))
	f.WriteString(startMarker)
	f.Write(content)
	f.WriteString("# ===== GitHub Hosts End =====\n")

	app.logWithLevel(SUCCESS, "hosts 文件更新成功")
	app.flushDNSCache()
	return nil
}
