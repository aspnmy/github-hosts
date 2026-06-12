package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// toggleAutoUpdate 切换自动更新状态
func (app *App) toggleAutoUpdate() error {
	config, err := app.loadConfig()
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	currentStatus := map[bool]string{true: "开启", false: "关闭"}[config.AutoUpdate]
	targetStatus := map[bool]string{true: "关闭", false: "开启"}[config.AutoUpdate]

	fmt.Printf("\n当前自动更新已%s，是否%s？[y/N]: ", currentStatus, targetStatus)

	var response string
	fmt.Scanf("%s", &response)

	if response != "y" && response != "Y" {
		app.logWithLevel(INFO, "保持当前状态不变")
		return nil
	}

	// 更新配置（保留原有时区设置）
	config.AutoUpdate = !config.AutoUpdate
	if err := app.updateConfig(config.UpdateInterval, config.AutoUpdate, config.TimeZone); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	if config.AutoUpdate {
		// 开启自动更新时，设置定时任务
		if err := app.setupCron(config.UpdateInterval); err != nil {
			app.logWithLevel(ERROR, "设置定时任务失败: %v", err)
			// 回滚配置
			config.AutoUpdate = false
			app.updateConfig(config.UpdateInterval, false, config.TimeZone)
			return fmt.Errorf("设置定时任务失败: %w", err)
		}
		app.logWithLevel(SUCCESS, "自动更新已开启，更新间隔为 %d 分钟", config.UpdateInterval)
	} else {
		// 关闭自动更新时，移除定时任务
		switch runtime.GOOS {
		case "darwin":
			exec.Command("launchctl", "bootout", "system/com.github.hosts").Run()
			os.Remove(darwinPlistPath)
		case "windows":
			exec.Command("schtasks", "/delete", "/tn", windowsTaskName, "/f").Run()
		default:
			os.Remove(linuxCronPath)
		}
		app.logWithLevel(SUCCESS, "自动更新已关闭")
	}

	return nil
}

// changeUpdateInterval 修改更新间隔
func (app *App) changeUpdateInterval() error {
	config, err := app.loadConfig()
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	fmt.Println("\n请选择新的更新间隔：")
	fmt.Println("1. 每 30 分钟")
	fmt.Println("2. 每 60 分钟")
	fmt.Println("3. 每 120 分钟")

	var choice int
	fmt.Scanf("%d", &choice)

	var interval int
	switch choice {
	case 1:
		interval = 30
	case 2:
		interval = 60
	case 3:
		interval = 120
	default:
		return fmt.Errorf("无效的选项")
	}

	// 更新配置（保留原有时区设置）
	if err := app.updateConfig(interval, config.AutoUpdate, config.TimeZone); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	// 如果启用了自动更新，则更新定时任务
	if config.AutoUpdate {
		if err := app.setupCron(interval); err != nil {
			return fmt.Errorf("更新定时任务失败: %w", err)
		}
	}

	app.logWithLevel(SUCCESS, "更新间隔已修改为 %d 分钟", interval)
	return nil
}

// detectSystemTimeZoneName 检测当前系统的本地时区名称（IANA 标准）
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 时区名称（如 Asia/Shanghai、America/New_York）；若无法获取则返回 "UTC"
//
// 说明：
//   先尝试读取系统本地时区的 name，若 Go 运行时无法解析则返回 UTC
func detectSystemTimeZoneName() string {
	// 从 time.Local 中获取时区名称
	name, _ := time.Now().Zone()
	if name != "" && name != "Local" {
		// 尝试加载该名称以验证是否为有效 IANA 名称
		if _, err := time.LoadLocation(name); err == nil {
			return name
		}
	}

	// 在常见的系统环境变量中读取
	for _, env := range []string{"TZ"} {
		if val := os.Getenv(env); val != "" {
			if _, err := time.LoadLocation(val); err == nil {
				return val
			}
		}
	}

	// 最后退回本地时区对象的名称
	if loc := time.Local.String(); loc != "Local" && loc != "" {
		return loc
	}

	// 兜底：使用 UTC
	return "UTC"
}

// isTimeZoneConfigured 判断配置中是否已经显式设置了时区
//
// 参数：
//   - tz: 配置中的时区字符串（来自 Config.TimeZone）
//
// 返回值：
//   - bool: 如果已设置（非空字符串）则返回 true
//
// 说明：
//   首次安装时 Config.TimeZone 为空字符串，此时用户需要确认时区；
//   后续再次运行安装向导时，已设置的时区划被视为已配置，直接引用即可
func isTimeZoneConfigured(tz string) bool {
	return tz != ""
}

// commonTimeZones 列出常见的时区供用户选择
//
// 参数：
//   - 无
//
// 返回值：
//   - []string: 常见 IANA 时区列表
func commonTimeZones() []string {
	return []string{
		"Asia/Shanghai",     // 中国标准时间（北京）
		"Asia/Hong_Kong",    // 香港
		"Asia/Tokyo",        // 日本
		"Asia/Singapore",    // 新加坡
		"Asia/Dubai",        // 阿联酋
		"Asia/Kolkata",      // 印度
		"Europe/London",     // 英国
		"Europe/Paris",      // 法国/德国
		"Europe/Moscow",     // 俄罗斯
		"America/New_York",  // 美国东部
		"America/Chicago",   // 美国中部
		"America/Los_Angeles", // 美国西部
		"America/Sao_Paulo", // 巴西
		"Australia/Sydney",  // 澳大利亚
		"UTC",               // 国际协调时间
	}
}

// promptTimeZoneSelection 显示常见时区列表供用户选择
//
// 参数：
//   - 无
//
// 返回值：
//   - string: 用户选择的 IANA 时区名称（选择无效时默认 Asia/Shanghai）
//
// 说明：
//   在安装向导（逻辑一）中，当用户选择「手动选择时区」时调用此函数
func promptTimeZoneSelection() string {
	zones := commonTimeZones()

	fmt.Println("\n请从以下常见时区中选择:")
	for i, z := range zones {
		fmt.Printf("  %d. %s\n", i+1, z)
	}
	fmt.Print("请输入选项: ")

	var choice int
	fmt.Scanf("%d", &choice)

	if choice < 1 || choice > len(zones) {
		fmt.Println("无效的选项，使用默认时区 Asia/Shanghai")
		return "Asia/Shanghai"
	}
	return zones[choice-1]
}

// changeTimeZone 在菜单中变更时区设置（用于已安装后修改时区）
//
// 参数：
//   - 无
//
// 返回值：
//   - error: 读取配置或写入配置失败时返回错误
//
// 说明：
//   1. 显示当前使用的时区
//   2. 提供用户重新检测系统时区、或手动选择其他时区的选项
//   3. 保存新的时区到配置文件
func (app *App) changeTimeZone() error {
	config, err := app.loadConfig()
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	fmt.Printf("\n当前配置的时区: %s\n", config.TimeZone)
	fmt.Printf("系统检测到的本地时区: %s\n", detectSystemTimeZoneName())
	fmt.Println("\n请选择时区设置方式:")
	fmt.Println("1. 使用系统检测到的本地时区")
	fmt.Println("2. 手动从常见时区中选择")

	var choice int
	fmt.Print("请输入选项 (1-2): ")
	fmt.Scanf("%d", &choice)

	newTZ := config.TimeZone
	switch choice {
	case 1:
		newTZ = detectSystemTimeZoneName()
	case 2:
		newTZ = promptTimeZoneSelection()
	default:
		return fmt.Errorf("无效的选项，时区保持不变")
	}

	// 验证时区是否有效
	if _, err := time.LoadLocation(newTZ); err != nil {
		return fmt.Errorf("时区 %s 无效: %w", newTZ, err)
	}

	if err := app.updateConfig(config.UpdateInterval, config.AutoUpdate, newTZ); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "时区已变更为: %s", newTZ)
	return nil
}

// getConfigTimeZone 从配置中获取时区；若未配置则使用系统本地时区
//
// 参数：
//   - app: App 实例，用于读取配置
//
// 返回值：
//   - *time.Location: 解析后的时区对象（可直接用于 time.In()）
//   - string:         时区名称（供显示/日志使用）
//
// 说明：
//   用于统一时间格式化和日志输出时的时区引用
func (app *App) getConfigTimeZone() (*time.Location, string) {
	config, err := app.loadConfig()
	var tzName string
	if err == nil && config != nil && config.TimeZone != "" {
		tzName = config.TimeZone
	} else {
		tzName = detectSystemTimeZoneName()
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.Local
	}
	return loc, tzName
}

// exportConfigToFile 导出配置到文件
func (app *App) exportConfigToFile() error {
	exportPath := filepath.Join(app.baseDir, fmt.Sprintf("config_export_%s.json", time.Now().Format("20060102_150405")))

	data, err := os.ReadFile(app.configFile)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("导出配置失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "配置已导出到: %s", exportPath)
	return nil
}

// importConfigFromFile 从文件导入配置
func (app *App) importConfigFromFile() error {
	fmt.Print("请输入配置文件路径: ")
	var path string
	fmt.Scanf("%s", &path)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("配置文件格式无效: %w", err)
	}

	if err := os.WriteFile(app.configFile, data, 0644); err != nil {
		return fmt.Errorf("更新配置失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "配置导入成功")
	return nil
}
