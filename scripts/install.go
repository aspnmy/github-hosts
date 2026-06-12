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

	// 首先检查 hosts 文件中是否已存在 GitHub Hosts 区块（基于 Start/End 标记）
	content, err := os.ReadFile(hostsFile)
	if err == nil && strings.Contains(string(content), hostsStartMarker) {
		// 已存在 GitHub Hosts 数据，询问是否更新
		updateResponse := promptString("\n检测到已存在 GitHub Hosts 数据，是否要更新？[Y/n]: ")

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
	response := promptString("\n是否开启自动更新？[Y/n]: ")

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
		choice, err := promptInt("请输入选项 (1-3): ")

		if err == nil {
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
		} else {
			app.logWithLevel(ERROR, "输入无效，将使用默认间隔（60分钟）")
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
		tzChoice, err := promptInt("请输入选项 (1-2): ")
		if err == nil && tzChoice == 2 {
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

// updateConfig 更新配置文件（用于 hosts 更新完成时，会同时更新 LastUpdate 时间戳）
//
// 参数：
//   - interval:   更新间隔（分钟）
//   - autoUpdate: 是否开启自动更新
//   - timeZone:   IANA 时区名称；为空时使用系统本地时区
//
// 返回值：
//   - error: 写入文件失败时返回错误
//
// 说明：
//   会更新 LastUpdate 为当前时间。如果只是修改配置偏好（如时区/开关），
//   请使用 updateConfigSettings 以保留真实的更新时间。
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
		LastUpdate:     time.Now().In(loc), // 仅在完整更新时刷新
		Version:        getAppVersion(),
		AutoUpdate:     autoUpdate,
		TimeZone:       timeZone,
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(app.configFile, data, 0644)
}

// updateConfigSettings 仅更新配置偏好（时区/自动更新/间隔），保留上次更新时间
//
// 参数：
//   - interval:   更新间隔（分钟）
//   - autoUpdate: 是否开启自动更新
//   - timeZone:   IANA 时区名称；为空时使用系统本地时区
//
// 返回值：
//   - error: 读取或写入文件失败时返回错误
//
// 说明：
//   用于「修改时区」、「切换自动更新」等仅修改配置偏好的场景，
//   不会重置 LastUpdate 字段，保证更新时间显示准确。
func (app *App) updateConfigSettings(interval int, autoUpdate bool, timeZone string) error {
	// 读取现有配置，保留 LastUpdate
	config, err := app.loadConfig()
	if err != nil {
		// 配置文件不存在，退化为完整更新（首次设置时）
		return app.updateConfig(interval, autoUpdate, timeZone)
	}

	if timeZone == "" {
		timeZone = detectSystemTimeZoneName()
	}

	// 保留 LastUpdate，仅覆盖其他字段
	config.UpdateInterval = interval
	config.AutoUpdate = autoUpdate
	config.TimeZone = timeZone

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
	// 步骤 1：备份当前 hosts 文件（独立步骤，失败不影响后续）
	app.logWithLevel(INFO, "开始备份当前 hosts 文件")
	if err := app.backupHosts(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	app.logWithLevel(SUCCESS, "hosts 文件备份完成")

	// 步骤 2：读取当前 hosts 文件内容，清理已有的 GitHub Hosts 区块
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("读取 hosts 文件失败: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, hostsStartMarker) {
			inBlock = true
			continue
		}
		if strings.Contains(trimmed, hostsEndMarker) {
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	// 去除末尾空行
	for len(filteredLines) > 0 && strings.TrimSpace(filteredLines[len(filteredLines)-1]) == "" {
		filteredLines = filteredLines[:len(filteredLines)-1]
	}
	filteredContent := strings.Join(filteredLines, "\n")
	if filteredContent != "" && !strings.HasSuffix(filteredContent, "\n") {
		filteredContent += "\n"
	}

	// 步骤 3：从服务器下载最新 hosts 数据
	app.logWithLevel(INFO, "正在从服务器获取最新 hosts 数据")
	resp, err := http.Get(hostsAPI)
	if err != nil {
		return fmt.Errorf("failed to download hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	newData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	app.logWithLevel(SUCCESS, "成功获取最新 hosts 数据")

	// 步骤 4：在内存中组装完整的新 hosts 文件内容
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		loc = time.Local
	}
	updateTime := time.Now().In(loc).Format(timeFormatStdTZ)

	var finalContent strings.Builder
	finalContent.WriteString(filteredContent)
	finalContent.WriteString("\n")
	finalContent.WriteString(hostsStartMarker)
	finalContent.WriteString(fmt.Sprintf(" \n# (Updated: %s, Timezone: %s)\n", updateTime, timeZone))
	finalContent.Write(newData)
	if !strings.HasSuffix(string(newData), "\n") {
		finalContent.WriteString("\n")
	}
	finalContent.WriteString(hostsEndMarker)
	finalContent.WriteString("\n")

	// 步骤 5：写入临时文件 → 原子替换（避免写入中途失败导致 hosts 文件损坏）
	tmpPath := hostsFile + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(finalContent.String()), 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("写入临时 hosts 文件失败: %w", err)
	}

	// 验证临时文件是否正确写入（至少应有 Start/End 标记）
	verifyData, err := os.ReadFile(tmpPath)
	if err != nil || !strings.Contains(string(verifyData), hostsStartMarker) ||
		!strings.Contains(string(verifyData), hostsEndMarker) {
		os.Remove(tmpPath)
		return fmt.Errorf("临时 hosts 文件校验失败，已中止更新")
	}

	// 原子替换：操作系统层面保证要么成功要么失败，不会出现部分写入
	if err := os.Rename(tmpPath, hostsFile); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("原子替换 hosts 文件失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "hosts 文件更新成功")

	// 步骤 6：刷新 DNS 缓存（非致命错误）
	app.logWithLevel(INFO, "正在刷新 DNS 缓存")
	if err := app.flushDNSCache(); err != nil {
		app.logWithLevel(WARNING, "DNS 缓存刷新失败: %v", err)
	} else {
		app.logWithLevel(SUCCESS, "DNS 缓存刷新完成")
	}

	return nil
}
