package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ==================== 统一输入处理 ====================
//
// 使用 bufio.Reader 逐行读取，避免 fmt.Scanf 与 fmt.Scanln 混用导致的换行符残留问题。

// stdinReader 全局标准输入读取器（跨调用保持状态）
var stdinReader = bufio.NewReader(os.Stdin)

// readLine 从标准输入读取一行（去掉末尾的换行符）
//
// 返回值：
//   - string: 读取到的一行内容（去掉 \r\n 或 \n）
//   - error:  读取失败时返回错误
func readLine() (string, error) {
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return strings.TrimSpace(line), err
	}
	return strings.TrimSpace(line), nil
}

// promptString 先打印提示再读取一行字符串（空输入不接受时会重复提示）
//
// 参数：
//   - prompt: 提示文本（不含末尾换行，会原样输出）
//
// 返回值：
//   - string: 用户输入的一行（已去空白）
func promptString(prompt string) string {
	fmt.Print(prompt)
	line, _ := readLine()
	return line
}

// promptInt 先打印提示，读取一行并解析为整数
//
// 参数：
//   - prompt: 提示文本
//
// 返回值：
//   - int: 解析成功的整数；失败时返回 -1
//   - error: 解析失败时返回错误（无错误时也可能用户输入为空）
func promptInt(prompt string) (int, error) {
	fmt.Print(prompt)
	line, err := readLine()
	if err != nil || line == "" {
		return -1, fmt.Errorf("未输入内容")
	}
	return strconv.Atoi(line)
}

// checkAndElevateSudo 检查权限并在需要时提权
func checkAndElevateSudo() error {
	// Windows 系统使用不同的权限检查方式
	if runtime.GOOS == "windows" {
		// 检查是否以管理员权限运行
		isAdmin, err := isWindowsAdmin()
		if err != nil {
			return fmt.Errorf("检查 Windows 权限失败: %w", err)
		}

		if !isAdmin {
			fmt.Println("需要管理员权限来修改 hosts 文件")
			fmt.Println("请右键点击程序，选择'以管理员身份运行'")

			// 获取当前可执行文件的路径
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("获取程序路径失败: %w", err)
			}

			// 使用 runas 命令提权运行
			cmd := exec.Command("powershell", "Start-Process", exe, "-Verb", "RunAs")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("提权失败: %w", err)
			}

			// 退出当前的非管理员进程
			os.Exit(0)
		}
		return nil
	}

	// Unix 系统的权限检查
	if os.Geteuid() == 0 {
		return nil
	}

	// 检查命令是否以 sudo 运行
	sudoUID := os.Getenv("SUDO_UID")
	if sudoUID != "" {
		return nil
	}

	fmt.Println("需要管理员权限来修改 hosts 文件")
	fmt.Println("请输入 sudo 密码：")

	// 获取当前可执行文件的路径
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	// 构建使用 sudo 运行的命令
	cmd := exec.Command("sudo", "-S", exe)

	// 将当前程序的标准输入输出连接到新进程
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 运行提权后的程序
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("提权失败: %w", err)
	}

	// 退出当前的非 root 进程
	os.Exit(0)
	return nil
}

// isWindowsAdmin 检查当前进程是否具有管理员权限
func isWindowsAdmin() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, fmt.Errorf("不是 Windows 系统")
	}

	// 创建一个测试文件在系统目录
	testPath := filepath.Join(os.Getenv("windir"), ".test")
	err := os.WriteFile(testPath, []byte("test"), 0644)
	if err == nil {
		// 如果成功创建，则删除测试文件
		os.Remove(testPath)
		return true, nil
	}

	// 如果创建失败，检查是否是权限问题
	if os.IsPermission(err) {
		return false, nil
	}

	return false, err
}

// clearScreen 清空控制台
func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default: // linux, darwin, etc
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func main() {
	// 检查权限并在需要时提权
	if err := checkAndElevateSudo(); err != nil {
		fmt.Printf("错误: %v\n", err)
		fmt.Println("请使用管理员权限运行此程序")
		os.Exit(1)
	}

	// 首次运行：确保 .version 文件存在（若不存在则写入当前版本号）
	_ = ensureVersionFile()

	clearScreen() // 启动时先清屏
	fmt.Print(getBanner())

	// 启动时静默检查是否有新版本
	runSilentUpdateCheck()

	app, err := NewApp()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	for {
		// 显示安装状态
		installed, _ := app.checkInstallStatus()
		app.displayInstallStatus()

		fmt.Println("\n[基础功能]")
		fmt.Println("1.  安装/更新")
		if installed {
			fmt.Println("2.  卸载程序")
			fmt.Println("3.  查看 hosts 内容")

			fmt.Println("\n[自动更新]")
			// 动态显示自动更新选项
			config, err := app.loadConfig()
			if err == nil {
				if config.AutoUpdate {
					fmt.Println("4.  关闭自动更新")
				} else {
					fmt.Println("4.  开启自动更新")
				}
			} else {
				fmt.Println("4.  切换自动更新")
			}
			fmt.Println("5.  修改更新间隔")

			fmt.Println("\n[系统工具]")
			fmt.Println("6.  测试网络连接")
			fmt.Println("7.  检查系统状态")
			fmt.Println("8.  查看更新日志")
			fmt.Println("9.  打开配置目录")
			fmt.Println("10. 打开配置文件")
			fmt.Println("11. 系统诊断")

			fmt.Println("\n[备份管理]")
			fmt.Println("12. 创建新备份")
			fmt.Println("13. 恢复备份")
			fmt.Println("14. 删除备份")

			fmt.Println("\n[配置管理]")
			fmt.Println("15. 导出配置")
			fmt.Println("16. 导入配置")
			fmt.Println("17. 时区设置")
		}

		fmt.Println("\n[程序更新]")
		fmt.Println("18. 检查程序更新")

		fmt.Println("\n[系统]")
		fmt.Println("19. 打开 hosts 文件")

		fmt.Println("\n[关于]")
		fmt.Println("20. 🐙 访问项目主页")

		fmt.Println("\n0.  退出程序")
		fmt.Printf("\n请输入选项 (0-20 或 q 退出): ")

		// 读取用户输入（使用 readLine 统一处理换行符）
		input, _ := readLine()

		// 检查是否是退出命令
		if input == "q" || input == "Q" {
			fmt.Println("感谢使用，再见！")
			return
		}

		// 转换输入为数字
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("无效的选项，请重试")
			waitForEnter()
			continue
		}

		// 在未安装状态下限制某些选项的访问
		if !installed && (choice >= 2 && choice <= 17) {
			fmt.Println("\n❌ 请先安装程序才能使用该功能")
			waitForEnter()
			continue
		}

		// 调用统一的菜单分发函数
		if app.dispatchChoice(choice, installed) {
			return // 用户选择了退出
		}
	}
}

// dispatchChoice 统一分发用户的菜单选择，返回 true 表示应退出程序
//
// 参数：
//   - choice: 用户选择的菜单项编号
//   - installed: 程序是否已安装（用于跳过未安装状态下不可用的选项）
//
// 返回值：
//   - bool: true 表示应退出程序，false 表示继续循环
func (app *App) dispatchChoice(choice int, installed bool) bool {
	switch choice {
	case 1: // 安装/更新
		if err := app.installMenu(); err != nil {
			app.logWithLevel(ERROR, "安装失败: %v", err)
		}
		waitForEnter()
	case 2: // 卸载
		if !installed {
			return false
		}
		if err := app.uninstall(); err != nil {
			app.logWithLevel(ERROR, "卸载失败: %v", err)
		}
		waitForEnter()
	case 3: // 查看 hosts
		if err := app.showHostsContent(); err != nil {
			app.logWithLevel(ERROR, "查看 hosts 内容失败: %v", err)
		}
		waitForEnter()
	case 4: // 切换自动更新
		if err := app.toggleAutoUpdate(); err != nil {
			app.logWithLevel(ERROR, "切换自动更新失败: %v", err)
		}
		waitForEnter()
	case 5: // 修改更新间隔
		if err := app.changeUpdateInterval(); err != nil {
			app.logWithLevel(ERROR, "修改更新间隔失败: %v", err)
		}
		waitForEnter()
	case 6: // 测试网络连接
		if err := app.testConnection(); err != nil {
			app.logWithLevel(ERROR, "网络测试失败: %v", err)
		}
		waitForEnter()
	case 7: // 检查系统状态
		if err := app.checkStatus(); err != nil {
			app.logWithLevel(ERROR, "状态检查失败: %v", err)
		}
		waitForEnter()
	case 8: // 查看更新日志
		if err := app.showUpdateLogs(); err != nil {
			app.logWithLevel(ERROR, "查看日志失败: %v", err)
		}
		waitForEnter()
	case 9: // 打开配置目录
		if err := app.openConfigDir(); err != nil {
			app.logWithLevel(ERROR, "打开配置目录失败: %v", err)
		}
		waitForEnter()
	case 10: // 打开配置文件
		if err := app.openConfigFile(); err != nil {
			app.logWithLevel(ERROR, "打开配置文件失败: %v", err)
		}
		waitForEnter()
	case 11: // 系统诊断
		if err := app.runDiagnostics(); err != nil {
			app.logWithLevel(ERROR, "系统诊断失败: %v", err)
		}
		waitForEnter()
	case 12: // 创建新备份
		if err := app.createNewBackup(); err != nil {
			app.logWithLevel(ERROR, "创建备份失败: %v", err)
		}
		waitForEnter()
	case 13: // 恢复备份
		if err := app.restoreBackupMenu(); err != nil {
			app.logWithLevel(ERROR, "恢复备份失败: %v", err)
		}
		waitForEnter()
	case 14: // 删除备份
		if err := app.deleteBackupMenu(); err != nil {
			app.logWithLevel(ERROR, "删除备份失败: %v", err)
		}
		waitForEnter()
	case 15: // 导出配置
		if err := app.exportConfigToFile(); err != nil {
			app.logWithLevel(ERROR, "导出配置失败: %v", err)
		}
		waitForEnter()
	case 16: // 导入配置
		if err := app.importConfigFromFile(); err != nil {
			app.logWithLevel(ERROR, "导入配置失败: %v", err)
		}
		waitForEnter()
	case 17: // 时区设置
		if err := app.changeTimeZone(); err != nil {
			app.logWithLevel(ERROR, "时区设置失败: %v", err)
		}
		waitForEnter()
	case 18: // 检查程序更新
		if err := runUpdateCheck(app); err != nil {
			app.logWithLevel(ERROR, "更新检查失败: %v", err)
		}
		waitForEnter()
	case 19: // 打开 hosts 文件
		if err := app.openHostsFile(); err != nil {
			app.logWithLevel(ERROR, "打开 hosts 文件失败: %v", err)
		}
		waitForEnter()
	case 20: // 访问项目主页
		if err := app.openGitHubRepo(); err != nil {
			app.logWithLevel(ERROR, "打开项目主页失败: %v", err)
		}
		waitForEnter()
	case 0: // 退出
		fmt.Println("感谢使用，再见！")
		return true
	default:
		fmt.Println("无效的选项，请重试")
		waitForEnter()
	}
	return false
}

// NewApp 创建新的应用实例
func NewApp() (*App, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".github-hosts")
	app := &App{
		baseDir:    baseDir,
		configFile: filepath.Join(baseDir, "config.json"),
		backupDir:  filepath.Join(baseDir, "backups"),
		logDir:     filepath.Join(baseDir, "logs"),
	}

	return app, nil
}

// openGitHubRepo 打开项目主页
func (app *App) openGitHubRepo() error {
	repoURL := "https://github.com/aspnmy/github-hosts"
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", repoURL)
	case "linux":
		cmd = exec.Command("xdg-open", repoURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", repoURL)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("打开浏览器失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "已在浏览器中打开项目主页")
	return nil
}

// loadConfig 加载配置文件，并对关键字段做合法性校验
//
// 参数：
//   - 无
//
// 返回值：
//   - *Config: 合法的配置对象指针；任何错误发生时返回 nil
//   - error: 读取/解析文件失败，或字段校验失败时返回错误
func (app *App) loadConfig() (*Config, error) {
	data, err := os.ReadFile(app.configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// 校验 UpdateInterval：允许的值为 30 / 60 / 120；0 视为未设置（自动更新未启用）
	// 其他值则回退为 60，保证后续定时任务调度不崩溃
	switch config.UpdateInterval {
	case 0, 30, 60, 120:
		// 合法值，通过
	default:
		config.UpdateInterval = 60
	}

	// 校验 TimeZone：空字符串时使用系统检测的时区
	if config.TimeZone == "" {
		config.TimeZone = detectSystemTimeZoneName()
	}

	return &config, nil
}

// waitForEnter 等待用户按回车并重新显示界面
func waitForEnter() {
	fmt.Print("\n按回车键继续...")
	readLine() // 读取一行，等待用户按回车
	clearScreen()
	fmt.Print(getBanner())
}

// checkInstallStatus 检查程序安装状态
// 只读一次 hosts 文件：同时判断是否已安装（含 Start 标记）和统计 GitHub 区块条目数
func (app *App) checkInstallStatus() (bool, *InstallStatus) {
	status := &InstallStatus{
		IsInstalled:    false,
		AutoUpdate:     false,
		UpdateInterval: 0,
		LastUpdate:     "",
		Version:        getAppVersion(),
		TimeZone:       "",
		HostsCount:     0,
	}

	// 读取配置（用于显示偏好设置，不作为"已安装"的判断依据）
	config, configErr := app.loadConfig()
	if configErr == nil && config != nil {
		status.AutoUpdate = config.AutoUpdate
		status.UpdateInterval = config.UpdateInterval
		status.TimeZone = config.TimeZone

		// 获取最后更新时间（按配置时区显示）
		loc, err := time.LoadLocation(config.TimeZone)
		if err != nil {
			loc = time.Local
		}
		if !config.LastUpdate.IsZero() {
			status.LastUpdate = config.LastUpdate.In(loc).Format(timeFormatStdTZ)
		}
	}

	// 一次读取 hosts 文件：同时判断是否已安装 + 统计 GitHub Hosts 区块条目数
	content, err := os.ReadFile(hostsFile)
	if err == nil {
		contentStr := string(content)
		// 判断是否已安装
		if strings.Contains(contentStr, hostsStartMarker) {
			status.IsInstalled = true
		}
		// 统计 GitHub Hosts 区块内的有效条目数
		inBlock := false
		for _, line := range strings.Split(contentStr, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, hostsStartMarker) {
				inBlock = true
				continue
			}
			if strings.Contains(trimmed, hostsEndMarker) {
				inBlock = false
				continue
			}
			if inBlock && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				status.HostsCount++
			}
		}
	}

	return status.IsInstalled, status
}

// displayInstallStatus 显示安装状态（使用 checkInstallStatus 中已缓存的数据）
func (app *App) displayInstallStatus() {
	installed, status := app.checkInstallStatus()

	fmt.Println("\n=== 系统状态 ===")
	if installed {
		fmt.Println("📦 安装状态: ✅ 已安装")
		fmt.Printf("🔄 自动更新: %s\n", formatBool(status.AutoUpdate))
		if status.AutoUpdate {
			fmt.Printf("⏱️  更新间隔: %d 分钟\n", status.UpdateInterval)
		}
		fmt.Printf("🕒 系统时区: %s\n", status.TimeZone)
		fmt.Printf("🕒 上次更新: %s\n", status.LastUpdate)
		fmt.Printf("📌 程序版本: %s\n", status.Version)
		fmt.Printf("📝 GitHub Hosts 记录数: %d\n", status.HostsCount)
	} else {
		fmt.Println("📦 安装状态: ❌ 未安装")
		fmt.Println("💡 提示: 请选择选项 1 进行安装")
	}
	fmt.Println(strings.Repeat("-", 30))
}

// formatBool 格式化布尔值显示
func formatBool(b bool) string {
	if b {
		return "✅ 已开启"
	}
	return "❌ 已关闭"
}

// InstallStatus 安装状态结构体
type InstallStatus struct {
	IsInstalled    bool
	AutoUpdate     bool
	UpdateInterval int
	LastUpdate     string
	Version        string
	TimeZone       string
	HostsCount     int // GitHub Hosts 条目数（由 checkInstallStatus 一次读取获得）
}

// openHostsFile 打开 hosts 文件
func (app *App) openHostsFile() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS 使用 open 命令
		cmd = exec.Command("open", hostsFile)
	case "linux":
		// Linux 使用 xdg-open 命令
		cmd = exec.Command("xdg-open", hostsFile)
	case "windows":
		// Windows 使用 notepad 打开
		cmd = exec.Command("notepad", hostsFile)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("打开 hosts 文件失败: %w", err)
	}

	app.logWithLevel(SUCCESS, "已打开 hosts 文件")
	return nil
}
