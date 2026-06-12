package main

import (
	"runtime"
	"time"
)

// ==================== 全局常量 ====================
// hosts 文件区块标记——用于识别 GitHub Hosts 相关条目
// 所有读写 hosts 文件的代码统一使用这些常量，避免硬编码字符串不一致
const (
	hostsStartMarker = "# ===== GitHub Hosts Start ====="
	hostsEndMarker   = "# ===== GitHub Hosts End ====="
)

// 通用时间格式常量——统一程序中所有时间显示
const (
	timeFormatStd    = "2006-01-02 15:04:05"         // 标准日期时间
	timeFormatStdTZ  = "2006-01-02 15:04:05 MST"     // 带时区的日期时间
	timeFormatFile   = "20060102"                    // 文件名用日期
	timeFormatFileTm = "20060102_150405"             // 文件名用日期+时间
)

// ==================== 数据结构 ====================

// App 应用程序结构体
type App struct {
	baseDir    string
	configFile string
	backupDir  string
	logDir     string
}

// Config 配置文件结构体
type Config struct {
	UpdateInterval int       `json:"updateInterval"`
	LastUpdate     time.Time `json:"lastUpdate"`
	Version        string    `json:"version"`
	AutoUpdate     bool      `json:"autoUpdate"`
	TimeZone       string    `json:"timezone"`
}

// LogLevel 定义日志级别
type LogLevel int

const (
	INFO LogLevel = iota
	SUCCESS
	WARNING
	ERROR
)

// 系统相关常量
var (
	// hostsAPI 定义 API 地址
	hostsAPI = "https://hosts.earth-online.org/hosts"

	// hostsFile 根据操作系统定义 hosts 文件路径
	hostsFile = getHostsFilePath()

	// 定时任务相关路径
	windowsTaskName = "GitHubHostsUpdate"
	darwinPlistPath = "/Library/LaunchDaemons/com.github.hosts.plist"
	linuxCronPath   = "/etc/cron.d/github-hosts"
)

// getHostsFilePath 根据操作系统返回 hosts 文件路径
func getHostsFilePath() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\System32\\drivers\\etc\\hosts"
	}
	return "/etc/hosts"
}
