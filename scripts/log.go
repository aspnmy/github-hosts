package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// 全局日志状态：持久化文件句柄，避免每次写入都 open/close
var (
	logFileHandle *os.File // 当前打开的日志文件句柄
	logFileDate   string   // 当前日志文件对应的日期（YYYYMMDD）
	logFileDir    string   // 当前日志目录，用于 MkdirAll 检查
)

// openLogFile 获取或打开日志文件句柄，按日期自动切换（跨午夜自动换文件）
//
// 参数：
//   - logDir: 日志目录路径（如 ~/.github-hosts/logs）
//
// 返回值：
//   - *os.File: 可写入的日志文件句柄（打开失败时返回 nil，仅控制台输出）
func openLogFile(logDir string) *os.File {
	currentDate := time.Now().Format(timeFormatFile)

	// 若已有句柄且日期未变，直接复用
	if logFileHandle != nil && currentDate == logFileDate && logDir == logFileDir {
		return logFileHandle
	}

	// 日期改变或首次调用——关闭旧句柄，打开新文件
	if logFileHandle != nil {
		logFileHandle.Close()
		logFileHandle = nil
	}

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("❌ 创建日志目录失败: %v\n", err)
		return nil
	}

	filePath := filepath.Join(logDir, fmt.Sprintf("update_%s.log", currentDate))
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("❌ 打开日志文件失败: %v\n", err)
		return nil
	}

	logFileHandle = f
	logFileDate = currentDate
	logFileDir = logDir
	return f
}

// logWithLevel 输出带有级别的日志，并同时写入日志文件
//
// 参数：
//   - level:  日志级别（INFO/SUCCESS/WARNING/ERROR）
//   - format: printf 风格的格式字符串
//   - args:   格式化参数
func (app *App) logWithLevel(level LogLevel, format string, args ...interface{}) {
	app.logWithLevelOpt(level, true, format, args...)
}

// logWithLevelOpt 输出带有级别的日志，可选择是否写入日志文件
//
// 参数：
//   - level:       日志级别
//   - writeToFile: 是否同时写入日志文件（true 时写入）
//   - format:      printf 风格的格式字符串
//   - args:        格式化参数
func (app *App) logWithLevelOpt(level LogLevel, writeToFile bool, format string, args ...interface{}) {
	var prefix string
	switch level {
	case INFO:
		prefix = "ℹ️  "
	case SUCCESS:
		prefix = "✅ "
	case WARNING:
		prefix = "⚠️  "
	case ERROR:
		prefix = "❌ "
	}

	timestamp := time.Now().Format(timeFormatStd)
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s[%s] %s\n", prefix, timestamp, message)

	// 输出到控制台
	fmt.Print(logLine)

	// 如果不需要写入文件，直接返回
	if !writeToFile {
		return
	}

	// 使用持久化句柄写入日志文件
	f := openLogFile(app.logDir)
	if f == nil {
		return
	}
	if _, err := f.WriteString(logLine); err != nil {
		fmt.Printf("❌ 写入日志失败: %v\n", err)
	}
}
