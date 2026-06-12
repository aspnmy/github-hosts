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

	// 首先检查是否已存在 hosts 数据
	content, err := os.ReadFile(hostsFile)
	if err == nil && strings.Contains(string(content), "GitHub Hosts") {
		// 已存在 GitHub Hosts 数据，询问是否更新
		fmt.Print("\n检测到已存在 GitHub Hosts 数据，是否要更新？[Y/n]: ")
		var updateResponse string
		fmt.Scanf("%s", &updateResponse)

		if updateResponse == "n" || updateResponse == "N" {
			app.logWithLevel(INFO, "已取消更新操作")
			return nil
		}

		// 用户选择更新，直接执行更新操作
		// 从配置中读取已设置的时区；若无配置，则使用系统本地时区
		cfg, _ := app.loadConfig()
		activeTimeZone := detectSystemTimeZoneName()
		if cfg != nil && cfg.TimeZone != "" {
			activeTimeZone = cfg.TimeZone
		}
		if err := app.updateHosts(activeTimeZone); err != nil {
			app.logWithLevel(ERROR, "更新 hosts 失败: %v", err)
			return fmt.Errorf("更新 hosts 失败: %w", err)
		}
		app.logWithLevel(SUCCESS, "hosts 文件更新完成（时区: %s）", activeTimeZone)
		return nil
	}

	// 不存在 GitHub Hosts 数据，执行完整的安装流程
	app.logWithLevel(INFO, "开始安装配置向导...")

	// 1. 选择是否开启自动更新
	var autoUpdate bool = true // 默认开启
	fmt.Print("\n是否开启自动更新？[Y/n]: ")
	var response string
	fmt.Scanf("%s", &response)

	var interval int = 60 // 默认 60 分钟
	if response == "n" || response == "N" {
		autoUpdate = false
		app.logWithLevel(INFO, "已禁用自动更新")
	} else {
		app.logWithLevel(INFO, "已启用自动更新")

		fmt.Println("\n请选择更新间隔：")
		fmt.Println("1. 每 30 分钟")
		fmt.Println("2. 每 60 分钟")
		fmt.Println("3. 每 120 分钟")
		fmt.Print("请输入选项 (1-3): ")

		var choice int
		fmt.Scanf("%d", &choice)

		switch choice {
		case 1:
			interval = 30
		case 2:
			interval = 60
		case 3:
			interval = 120
		default:
			app.logWithLevel(ERROR, "无效的选项，将使用默认间隔（60分钟）")
			interval = 60
		}
		app.logWithLevel(INFO, "选择的更新间隔: %d 分钟", interval)
	}

	app.logWithLevel(INFO, "开始执行安装流程...")

	// 1. Setup directories
	app.logWithLevel(INFO, "第 1/4 步: 创建必要的目录结构")
	if err := app.setupDirectories(); err != nil {
		app.logWithLevel(ERROR, "创建目录失败: %v", err)
		return fmt.Errorf("创建目录失败: %w", err)
	}
	app.logWithLevel(SUCCESS, "目录创建完成")
	app.logWithLevel(INFO, "  - 基础目录: %s", app.baseDir)
	app.logWithLevel(INFO, "  - 配置文件: %s", app.configFile)
	app.logWithLevel(INFO, "  - 备份目录: %s", app.backupDir)
	app.logWithLevel(INFO, "  - 日志目录: %s", app.logDir)

	// 2. Update config
	app.logWithLevel(INFO, "第 2/5 步: 更新配置文件")

	// 加载现有配置（用于判断是否已经设置过时区）
	config, _ := app.loadConfig()
	existingTimeZone := ""
	if config != nil {
		existingTimeZone = config.TimeZone
	}

	// 检测并设置时区
	systemTZ := detectSystemTimeZoneName()
	app.logWithLevel(INFO, "检测到系统时区: %s", systemTZ)

	selectedTZ := systemTZ
	if !isTimeZoneConfigured(existingTimeZone) {
		// 逻辑一：客户端未设置过时区或使用默认时区，需要用户确认
		app.logWithLevel(INFO, "尚未在配置中设置时区，需要您确认时区信息")
		fmt.Printf("\n当前检测到的系统时区为: %s\n", systemTZ)
		fmt.Println("请选择时区设置方式:")
		fmt.Println("1. 使用系统检测到的时区（推荐）")
		fmt.Println("2. 手动选择常见时区")
		fmt.Print("请输入选项 (1-2): ")

		var tzChoice int
		fmt.Scanf("%d", &tzChoice)

		if tzChoice == 2 {
			selectedTZ = promptTimeZoneSelection()
		}
		app.logWithLevel(INFO, "已选择时区: %s", selectedTZ)
	} else {
		// 逻辑二：客户端已经配置了时区，直接引用
		selectedTZ = existingTimeZone
		app.logWithLevel(INFO, "使用配置文件中的时区: %s", selectedTZ)
	}

	if err := app.updateConfig(interval, autoUpdate, selectedTZ); err != nil {
		app.logWithLevel(ERROR, "更新配置失败: %v", err)
		return fmt.Errorf("更新配置失败: %w", err)
	}
	app.logWithLevel(SUCCESS, "配置文件更新完成（时区: %s）", selectedTZ)

	// 3. Update hosts
	app.logWithLevel(INFO, "第 3/5 步: 更新 hosts 文件")
	if err := app.updateHosts(selectedTZ); err != nil {
		app.logWithLevel(ERROR, "更新 hosts 失败: %v", err)
		return fmt.Errorf("更新 hosts 失败: %w", err)
	}
	app.logWithLevel(SUCCESS, "hosts 文件更新完成")

	// 4. Setup cron
	if autoUpdate {
		app.logWithLevel(INFO, "第 4/5 步: 设置定时更新任务")
		if err := app.setupCron(interval); err != nil {
			app.logWithLevel(ERROR, "设置定时任务失败: %v", err)
			return fmt.Errorf("设置定时任务失败: %w", err)
		}
		app.logWithLevel(SUCCESS, "定时任务设置完成")
	} else {
		app.logWithLevel(INFO, "已跳过定时任务设置（自动更新已禁用）")
	}

	// 5. 显示安装完成信息
	app.logWithLevel(SUCCESS, "安装完成！")
	app.logWithLevel(INFO, "系统配置信息：")
	if autoUpdate {
		app.logWithLevel(INFO, "  • 更新间隔: 每 %d 分钟", interval)
	}
	app.logWithLevel(INFO, "  • 自动更新: %s", map[bool]string{true: "已启用", false: "已禁用"}[autoUpdate])
	app.logWithLevel(INFO, "  • 当前时区: %s", selectedTZ)
	app.logWithLevel(INFO, "  • 配置文件: %s", app.configFile)
	app.logWithLevel(INFO, "  • 日志文件: %s", filepath.Join(app.logDir, "update.log"))
	app.logWithLevel(INFO, "  • 备份目录: %s", app.backupDir)

	// 显示当前 hosts 文件内容
	app.logWithLevel(INFO, "\n当前 hosts 文件内容：")
	fmt.Println("----------------------------------------")
	content, err = os.ReadFile(hostsFile)
	if err != nil {
		app.logWithLevel(ERROR, "读取 hosts 文件失败: %v", err)
	} else {
		fmt.Println(string(content))
	}
	fmt.Println("----------------------------------------")

	// 自动执行网络连接测试
	app.logWithLevel(INFO, "\n开始测试网络连接...")
	if err := app.testConnection(); err != nil {
		app.logWithLevel(WARNING, "网络连接测试出现问题: %v", err)
	}

	return nil
}

func (app *App) setupDirectories() error {
	dirs := []string{app.baseDir, app.backupDir, app.logDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// updateConfig 更新配置文件
//
// 参数：
//   - interval:   自动更新间隔，单位分钟
//   - autoUpdate: 是否开启自动更新
//   - timeZone:   IANA 时区名称（如 Asia/Shanghai、America/New_York），为空则使用系统本地时区
//
// 返回值：
//   - error: 写入文件失败时返回错误
func (app *App) updateConfig(interval int, autoUpdate bool, timeZone string) error {
	// 若未提供时区，使用系统本地时区名称
	if timeZone == "" {
		timeZone = detectSystemTimeZoneName()
	}

	// 将 LastUpdate 时间转换到配置的时区
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		loc = time.Local
	}

	config := Config{
		UpdateInterval: interval,
		LastUpdate:     time.Now().In(loc),
		Version:        "1.0.0",
		AutoUpdate:     autoUpdate,
		TimeZone:       timeZone,
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(app.configFile, data, 0644)
}

// updateHosts 从服务器获取最新 hosts 数据并更新本地 hosts 文件
//
// 参数：
//   - timeZone: IANA 时区名称，用于在 hosts 文件中显示更新时间戳
//
// 返回值：
//   - error: 更新失败时返回错误，包含具体阶段信息（备份/清理/下载/写入/刷新 DNS）
func (app *App) updateHosts(timeZone string) error {
	app.logWithLevel(INFO, "开始备份当前 hosts 文件")
	if err := app.backupHosts(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	app.logWithLevel(SUCCESS, "hosts 文件备份完成")

	// 先清理已存在的 GitHub Hosts 内容
	app.logWithLevel(INFO, "清理已存在的 GitHub Hosts 内容")
	if err := app.cleanHostsFile(); err != nil {
		return fmt.Errorf("清理已存在内容失败: %w", err)
	}
	app.logWithLevel(SUCCESS, "已清理旧的 hosts 内容")

	app.logWithLevel(INFO, "正在从服务器获取最新 hosts 数据")
	resp, err := http.Get(hostsAPI)
	if err != nil {
		return fmt.Errorf("failed to download hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	app.logWithLevel(SUCCESS, "成功获取最新 hosts 数据")

	app.logWithLevel(INFO, "正在更新本地 hosts 文件")
	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open hosts file: %w", err)
	}
	defer f.Close()

	// 根据配置时区显示更新时间
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		loc = time.Local
	}
	updateTime := time.Now().In(loc).Format("2006-01-02 15:04:05 MST")

	startMarker := fmt.Sprintf("\n# ===== GitHub Hosts Start ===== \n# (Updated: %s, Timezone: %s)\n",
		updateTime, timeZone)
	if _, err := f.WriteString(startMarker); err != nil {
		return fmt.Errorf("failed to write start marker: %w", err)
	}

	// 写入 hosts 内容
	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("failed to write hosts content: %w", err)
	}

	// 添加结束标记
	endMarker := "# ===== GitHub Hosts End =====\n"
	if _, err := f.WriteString(endMarker); err != nil {
		return fmt.Errorf("failed to write end marker: %w", err)
	}

	app.logWithLevel(SUCCESS, "hosts 文件更新成功")

	app.logWithLevel(INFO, "正在刷新 DNS 缓存")
	if err := app.flushDNSCache(); err != nil {
		app.logWithLevel(WARNING, "DNS 缓存刷新失败: %v", err)
	} else {
		app.logWithLevel(SUCCESS, "DNS 缓存刷新完成")
	}

	return nil
}
