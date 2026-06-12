package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

// hostTest 表示一个待测试的域名条目
type hostTest struct {
	name string
	ip   string
	host string
}

// hostTestResult 保存单个测试的结果
type hostTestResult struct {
	index    int
	host     string
	expected string
	actualIP string
	status   string
	elapsed  time.Duration
	dnsErr   error
	httpErr  error
}

// testConnection 测试网络连接（并发执行，所有域名同时请求 DNS 和 HTTP）
func (app *App) testConnection() error {
	app.logWithLevel(INFO, "开始网络连接测试...")
	fmt.Println("\n=== 连接测试结果 ===")

	// 读取 hosts 文件
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("\n❌ 严重错误：无法读取 hosts 文件\n")
		fmt.Printf("❌ 错误详情：%v\n", err)
		return err
	}

	// 解析 hosts 文件中的 GitHub 相关记录
	var tests []hostTest

	startMarker := "# ===== GitHub Hosts Start ====="
	endMarker := "# ===== GitHub Hosts End ====="
	inGithubSection := false
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == startMarker {
			inGithubSection = true
			continue
		}
		if line == endMarker {
			inGithubSection = false
			continue
		}

		if inGithubSection && line != "" && !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				tests = append(tests, hostTest{
					name: fields[1],
					ip:   fields[0],
					host: fields[1],
				})
			}
		}
	}

	if len(tests) == 0 {
		fmt.Printf("\n❌ 错误：在 hosts 文件中未找到 GitHub 相关记录\n")
		return fmt.Errorf("no github hosts found")
	}

	// 创建 HTTP 客户端（net/http Client 是并发安全的）
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			DisableKeepAlives:     true,
		},
	}

	// 并发执行所有测试，最多 10 个同时进行
	const maxConcurrency = 10
	results := make([]hostTestResult, len(tests))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	for i, test := range tests {
		wg.Add(1)
		// 按值捕获循环变量，避免 goroutine 闭包问题
		go func(idx int, t hostTest) {
			defer wg.Done()

			// 获取信号量：限制同时运行的请求数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			actualIP := ""
			status := "✓ 正常"
			var dnsErr, httpErr error
			failed := false

			// 获取实际 DNS 解析结果
			addrs, err := net.LookupHost(t.host)
			if err != nil {
				status = "✗ DNS失败"
				dnsErr = err
				failed = true
			} else if len(addrs) > 0 {
				actualIP = addrs[0]
			}

			// 测试 HTTP 连接（DNS 失败也可以跳过 HTTP）
			if !failed {
				resp, err := client.Get("https://" + t.host)
				if err != nil {
					status = "✗ 连接失败"
					httpErr = err
					failed = true
				} else {
					resp.Body.Close() // 立即关闭，不在 defer 中累积
					if actualIP != t.ip {
						status = "! IP不匹配"
						failed = true
					} else if resp.StatusCode != http.StatusOK {
						status = fmt.Sprintf("! 状态%d", resp.StatusCode)
						failed = true
					}
				}
			}

			results[idx] = hostTestResult{
				index:    idx,
				host:     t.host,
				expected: t.ip,
				actualIP: actualIP,
				status:   status,
				elapsed:  time.Since(start),
				dnsErr:   dnsErr,
				httpErr:  httpErr,
			}
		}(i, test)
	}

	// 等待所有测试完成
	wg.Wait()

	// 使用 tabwriter 创建表格（按原始顺序输出，保证结果一致）
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "\n%s\t%s\t%s\t%s\t%s\n",
		"域名",
		"状态",
		"响应时间",
		"当前解析IP",
		"期望IP")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	// 统计与输出
	successCount := 0
	failCount := 0

	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.host,
			r.status,
			fmt.Sprintf("%.2fs", r.elapsed.Seconds()),
			r.actualIP,
			r.expected)

		if r.dnsErr != nil {
			fmt.Printf("❌ %s DNS 解析失败: %v\n", r.host, r.dnsErr)
			failCount++
		} else if r.httpErr != nil {
			fmt.Printf("❌ %s 连接失败: %v\n", r.host, r.httpErr)
			failCount++
		} else if strings.HasPrefix(r.status, "!") {
			if r.actualIP != r.expected {
				fmt.Printf("⚠️  %s 的 IP 不匹配！当前: %s, 期望: %s\n", r.host, r.actualIP, r.expected)
			}
			failCount++
		} else {
			successCount++
		}
	}

	fmt.Fprintln(w, strings.Repeat("-", 100))
	w.Flush()

	// 输出总结
	fmt.Printf("\n测试总结:\n")
	fmt.Printf("总计测试: %d\n", len(tests))
	if successCount > 0 {
		fmt.Printf("✅ 成功: %d\n", successCount)
	}
	if failCount > 0 {
		fmt.Printf("❌ 失败: %d\n", failCount)
		fmt.Printf("\n⚠️  警告：检测到 %d 个问题，建议重新执行更新操作\n", failCount)
	} else {
		fmt.Printf("\n✅ 太好了！所有测试都通过了\n")
	}

	return nil
}

// flushDNSCache 刷新 DNS 缓存
func (app *App) flushDNSCache() error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("killall", "-HUP", "mDNSResponder").Run()
	case "linux":
		if err := exec.Command("systemd-resolve", "--flush-caches").Run(); err != nil {
			return exec.Command("systemctl", "restart", "systemd-resolved").Run()
		}
		return nil
	case "windows":
		return exec.Command("ipconfig", "/flushdns").Run()
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}
