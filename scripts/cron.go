package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

func (app *App) setupCron(interval int) error {
	scriptPath := filepath.Join(app.baseDir, "update.sh")
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(app.baseDir, "update.bat")
	}
	if err := app.createUpdateScript(scriptPath); err != nil {
		return fmt.Errorf("创建更新脚本失败: %w", err)
	}
	switch runtime.GOOS {
	case "darwin": return app.setupDarwinCron(interval, scriptPath)
	case "linux": return app.setupLinuxCron(interval, scriptPath)
	case "windows": return app.setupWindowsCron(interval, scriptPath)
	default: return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (app *App) createUpdateScript(path string) error {
	var content string
	// 在脚本中使用 loadHostSource 方式获取URL
	if runtime.GOOS == "windows" {
		content = `@echo off
echo [%date% %time%] 开始更新 >> {{.LogDir}}\update.log
powershell -Command "& {(Get-Content '{{.HostsFile}}') -notmatch '===== GitHub Hosts (Start|End) =====' | Set-Content '{{.HostsFile}}.tmp'}"
move /Y "{{.HostsFile}}.tmp" "{{.HostsFile}}"
echo # ===== GitHub Hosts Start ===== >> "{{.HostsFile}}"
powershell -Command "& {(New-Object System.Net.WebClient).DownloadString('{{.HostsAPI}}')}" >> "{{.HostsFile}}"
echo # ===== GitHub Hosts End ===== >> "{{.HostsFile}}"
ipconfig /flushdns
echo [%date% %time%] 完成 >> {{.LogDir}}\update.log
`
	} else {
		content = `#!/bin/bash
CONFIG_FILE="$HOME/.aspnmy/hostsource.json"
API_URL=$(python3 -c "
import json
with open('$CONFIG_FILE') as f: cfg = json.load(f)
src = cfg['sources'].get(cfg['source'], {})
print(src.get('hosts_url', '{{.HostsAPI}}'))
" 2>/dev/null || echo '{{.HostsAPI}}')
sed -i.bak '/# ===== GitHub Hosts Start =====/,/# ===== GitHub Hosts End =====/d' {{.HostsFile}}
echo "# ===== GitHub Hosts Start =====" >> {{.HostsFile}}
curl -fsSL "$API_URL" >> {{.HostsFile}}
echo "# ===== GitHub Hosts End =====" >> {{.HostsFile}}
[ "$(uname)" == "Darwin" ] && killall -HUP mDNSResponder || systemd-resolve --flush-caches 2>/dev/null || true
`
	}
	data := struct {
		LogDir, HostsFile, HostsAPI string
	}{LogDir: app.logDir, HostsFile: hostsFile, HostsAPI: hostsAPI}
	tmpl, _ := template.New("script").Parse(content)
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	defer f.Close()
	return tmpl.Execute(f, data)
}

func (app *App) setupWindowsCron(interval int, scriptPath string) error {
	exec.Command("schtasks", "/delete", "/tn", "GitHubHostsUpdate", "/f").Run()
	cmd := exec.Command("schtasks", "/create", "/tn", "GitHubHostsUpdate",
		"/tr", scriptPath, "/sc", "minute", "/mo", fmt.Sprintf("%d", interval), "/ru", "SYSTEM", "/f")
	out, err := cmd.CombinedOutput()
	if err != nil { return fmt.Errorf("创建计划任务失败: %s, %v", string(out), err) }
	return nil
}

func (app *App) setupDarwinCron(interval int, scriptPath string) error {
	exec.Command("launchctl", "bootout", "system/com.github.hosts").Run()
	os.Remove("/Library/LaunchDaemons/com.github.hosts.plist")
	content := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.github.hosts</string>
<key>ProgramArguments</key><array><string>/bin/bash</string><string>%s</string></array>
<key>StartInterval</key><integer>%d</integer>
<key>RunAtLoad</key><true/>
</dict></plist>`, scriptPath, interval*60)
	os.WriteFile("/Library/LaunchDaemons/com.github.hosts.plist", []byte(content), 0644)
	exec.Command("launchctl", "bootstrap", "system", "/Library/LaunchDaemons/com.github.hosts.plist").Run()
	return nil
}

func (app *App) setupLinuxCron(interval int, scriptPath string) error {
	schedules := map[int]string{30: "*/30 * * * *", 60: "0 * * * *", 120: "0 */2 * * *"}
	schedule, ok := schedules[interval]
	if !ok { return fmt.Errorf("无效间隔: %d", interval) }
	os.WriteFile("/etc/cron.d/github-hosts", []byte(fmt.Sprintf("%s root %s >/dev/null 2>&1\n", schedule, scriptPath)), 0644)
	exec.Command("systemctl", "restart", "cron").Run()
	return nil
}
