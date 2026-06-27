package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type App struct {
	baseDir    string
	configFile string
	backupDir  string
	logDir     string
	logger     *log.Logger
}

type Config struct {
	UpdateInterval int       `json:"updateInterval"`
	LastUpdate     time.Time `json:"lastUpdate"`
	Version        string    `json:"version"`
	AutoUpdate     bool      `json:"autoUpdate"`
}

// HostSource 数据源配置，从 ~/.aspnmy/hostsource.json 读取
type HostSource struct {
	Source  string              `json:"source"`
	Sources map[string]SourceDef `json:"sources"`
}

type SourceDef struct {
	Name     string `json:"name"`
	HostsURL string `json:"hosts_url"`
	Enabled  bool   `json:"enabled"`
}

type LogLevel int

const (
	INFO LogLevel = iota
	SUCCESS
	WARNING
	ERROR
)

var (
	hostsAPI  = "https://hosts.earth-online.org/hosts"
	hostsFile = getHostsFilePath()
)

func getHostsFilePath() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\System32\\drivers\\etc\\hosts"
	}
	return "/etc/hosts"
}

// loadHostSource 读取 ~/.aspnmy/hostsource.json，返回当前启用的数据源URL
func loadHostSource() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return hostsAPI
	}
	cfgPath := filepath.Join(home, ".aspnmy", "hostsource.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return hostsAPI
	}
	var hs HostSource
	if err := json.Unmarshal(data, &hs); err != nil {
		return hostsAPI
	}
	src, ok := hs.Sources[hs.Source]
	if !ok || !src.Enabled || src.HostsURL == "" {
		return hostsAPI
	}
	fmt.Printf("  数据源: %s (%s)\n", hs.Source, src.Name)
	fmt.Printf("  URL: %s\n", src.HostsURL)
	return src.HostsURL
}

// 定时任务相关路径（从原始main.go移过来）
var (
	windowsTaskName = "GitHubHostsUpdate"
	darwinPlistPath = "/Library/LaunchDaemons/com.github.hosts.plist"
	linuxCronPath   = "/etc/cron.d/github-hosts"
)
