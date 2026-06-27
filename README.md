<div align="center">
  <img src="public/logo.svg" width="140" height="140" alt="github-hosts logo">
  <h1>github-hosts (NoKV)</h1>
</div>

> 本项目是基于 [TinsFox/github-hosts](https://github.com/TinsFox/github-hosts) 的 **main** 分支，移除了 Cloudflare KV 存储依赖，每次请求实时 DNS 查询获取最新 IP 记录。
>
> **为什么要移除 KV？** 原版使用 Cloudflare KV 存储数据，但 KV 免费配额容易耗尽（被刷），导致服务不可用。本版本彻底移除 KV，无需 KV 配额，永不下线。
> 客户端每次访问接口都实时更新一下hosts地址列表，移除KV存储依赖，完全开源免费，永不下线。
>
> 在线地址：[https://hosts.earth-online.org](https://hosts.earth-online.org)
>
> 克隆 NoKV 版本：`git clone -b nokv https://github.com/aspnmy/github-hosts.git`
>
> 原作者仓库(有KV依赖)：[https://github.com/TinsFox/github-hosts](https://github.com/TinsFox/github-hosts)

## 特性

- 🚀 使用 [Hono](https://hono.dev/) + Cloudflare Workers 部署，无需自有服务器
- 🌍 多 DNS 服务支持（Cloudflare `1.1.1.1`、Google `dns.google`）
- ⚡️ **实时 DNS 查询**：每次请求直接查 DNS over HTTPS，不依赖缓存存储
- 🔄 **多种使用方式**：SwitchHosts / 手动 / Shell 脚本 / Go 客户端程序
- 📡 **完整 REST API**：提供 `/hosts`、`/hosts.json`、`/{domain}`、`/reset` 接口
- 🚫 **彻底移除 KV**：不消耗 Cloudflare KV 配额，永不下线
- 📝 **动态域名配置**：通过 `domains.txt` 自定义支持的域名列表，5 分钟运行时缓存
- 💾 **GitHub Actions 自动发布**：每次打 tag 自动编译 7 个平台的二进制 + Shell 脚本 ZIP
- 🛡️ **限流保护**：域名查询 30 req/min，管理接口 5 req/min
- ⚙️ **交互式 Go 客户端**：18 项功能菜单（备份/恢复/诊断/自更新等）
- 🐚 **跨平台 Shell 脚本**：适用于所有类 Unix 系统（含软路由、嵌入式）

## 使用方法

### 1. SwitchHosts 工具

1. 下载 [SwitchHosts](https://github.com/oldj/SwitchHosts)
2. 添加规则：
   - 方案名：GitHub Hosts
   - 类型：远程
   - URL：`https://hosts.earth-online.org/hosts`
   - 自动更新：1 小时

### 2. 手动更新

1. 获取 hosts：访问 [https://hosts.earth-online.org/hosts](https://hosts.earth-online.org/hosts)
2. 更新本地 hosts 文件：
   - Windows：`C:\Windows\System32\drivers\etc\hosts`
   - MacOS/Linux：`/etc/hosts`
3. 刷新 DNS：
   - Windows：`ipconfig /flushdns`
   - MacOS：`sudo killall -HUP mDNSResponder`
   - Linux：`sudo systemd-resolve --flush-caches`

<h4>Windows 用户</h4>
  <p>在管理员权限的 PowerShell 中执行（以最新 Release 的 <code>github-hosts.windows-amd64.exe.zip</code> 为例）：</p>
  <pre><code class="language-powershell"># 下载最新 Release 中的 zip 包并解压
$zip = "$env:TEMP\github-hosts.zip"
Invoke-WebRequest -Uri "https://github.com/aspnmy/github-hosts/releases/latest/download/github-hosts.windows-amd64.exe.zip" -OutFile $zip
Expand-Archive -Path $zip -DestinationPath "$env:TEMP\github-hosts" -Force
# 以管理员身份运行
&amp; "$env:TEMP\github-hosts\github-hosts.windows-amd64.exe"</code></pre>

  <h4>Linux 用户</h4>
  <pre><code class="language-bash">curl -L https://github.com/aspnmy/github-hosts/releases/latest/download/github-hosts.linux-amd64.zip -o github-hosts.zip
unzip github-hosts.zip
chmod +x github-hosts.linux-amd64
sudo ./github-hosts.linux-amd64</code></pre>

  <p>更多版本请查看 <a href="https://github.com/aspnmy/github-hosts/releases">Release 页面</a>（包含 Windows/Linux/macOS 的 32/64/ARM 版本，以及通用的 Shell 脚本 <code>github_hosts.sh.zip</code>）。</p>

### 3. Shell 脚本（通用，适用于所有类 Unix 系统）

`github_hosts.sh` 是一个轻量级的 Bash 脚本，**不依赖 Go 运行时，不依赖平台架构**，适用于任何 Linux / macOS 以及一些高度定制化的类 Unix 系统（例如绿联云 DX4600 未升级版本、某些旧版嵌入式 Linux、容器环境、软路由系统等）。

它会从 `https://hosts.earth-online.org/hosts` 实时下载最新的 hosts，提取 GitHub 相关的区块，并安全地写入到本地 `/etc/hosts`。

#### 获取方式

- 在每个 Release 的 Assets 中都会附带 `github_hosts.sh.zip`，解压后得到 `github_hosts.sh`
- 也可以直接从仓库克隆：`git clone -b nokv https://github.com/aspnmy/github-hosts.git`，脚本位于 `scripts/github_hosts.sh`

#### 使用方式

```bash
# 1. 下载并解压（替换为最新 Release 的 github_hosts.sh.zip 地址）
curl -L https://github.com/aspnmy/github-hosts/releases/latest/download/github_hosts.sh.zip -o github_hosts.sh.zip
unzip github_hosts.sh.zip

# 2. 赋予执行权限
chmod +x github_hosts.sh

# 3. 以 root 身份运行（会自动备份当前 /etc/hosts，替换 GitHub hosts 区块，刷新 DNS 缓存）
sudo ./github_hosts.sh
```

#### 定时自动更新（Linux / macOS）

```bash
# 加到 cron，每天凌晨 2:30 自动更新
sudo crontab -e
# 添加以下一行（路径根据实际位置调整）
# 30 2 * * * /path/to/github_hosts.sh >> /var/log/github_hosts.log 2>&1
```

#### 脚本行为说明

1. 检查是否以 root/sudo 身份运行
2. 从 `https://hosts.earth-online.org/hosts` 下载最新内容
3. 提取 `# github hosts` 到「更新时间」之间的区块
4. 将当前 `/etc/hosts` 备份到 `/etc/hosts.backup/hosts_日期`
5. 如果本地已有旧的 GitHub hosts 区块，则替换；没有则追加到末尾
6. 尝试刷新系统 DNS 缓存（`systemd-resolve` / `resolvectl` / `nscd`）
7. 临时文件通过 `trap EXIT` 自动清理

### 4. Go 客户端功能菜单说明

运行编译后的 `github-hosts` 程序会进入一个交互式的文本菜单（需管理员 / root 权限）。下面是各选项的说明（当前菜单共 20 项 + 退出）：

| 分区 | 选项 | 功能 | 对应实现文件 / 函数 |
|------|------|------|-------------------|
| 基础功能 | 1 | **安装 / 更新**：下载最新 GitHub hosts 并合并到系统 hosts 文件。首次运行即为安装，后续会更新原有区块。 | [install.go](file:///v:/git_data/github-hosts/scripts/install.go) `installMenu()` / `updateHosts()` |
| 基础功能 | 2 | **卸载程序**：清除本程序写入 hosts 的 GitHub 区块，并移除配置目录与定时任务。 | [uninstall.go](file:///v:/git_data/github-hosts/scripts/uninstall.go) `uninstall()` / `cleanHostsFile()` |
| 基础功能 | 3 | **查看 hosts 内容**：在终端显示当前系统 hosts 文件中的 GitHub 相关解析条目。 | [menu.go](file:///v:/git_data/github-hosts/scripts/menu.go) `showHostsContent()` |
| 自动更新 | 4 | **开启 / 关闭自动更新**：根据当前状态动态切换。为当前用户注册后台定时任务（Windows 任务计划程序，Linux/macOS cron）。 | [config.go](file:///v:/git_data/github-hosts/scripts/config.go) `toggleAutoUpdate()` |
| 自动更新 | 5 | **修改更新间隔**：在 30 / 60 / 120 分钟三档中选择。 | [config.go](file:///v:/git_data/github-hosts/scripts/config.go) `changeUpdateInterval()` |
| 系统工具 | 6 | **测试网络连接**：并发测试 `github.com`、`hosts.earth-online.org` 等关键域名的 DNS 与 HTTPS 可达性。 | [network.go](file:///v:/git_data/github-hosts/scripts/network.go) `testConnection()` |
| 系统工具 | 7 | **检查系统状态**：显示安装状态、当前版本号（来自可执行文件所在目录的 `.version`）、hosts 区块条目数量、配置目录路径与定时任务状态。 | [menu.go](file:///v:/git_data/github-hosts/scripts/menu.go) `checkStatus()` / [main.go](file:///v:/git_data/github-hosts/scripts/main.go) `displayInstallStatus()` |
| 系统工具 | 8 | **查看更新日志**：读取本程序日志目录下的运行日志，供诊断问题。 | [menu.go](file:///v:/git_data/github-hosts/scripts/menu.go) `showUpdateLogs()` |
| 系统工具 | 9 | **打开配置目录**：定位并打开本程序生成的配置文件目录。 | [menu.go](file:///v:/git_data/github-hosts/scripts/menu.go) `openConfigDir()` |
| 系统工具 | 10 | **打开配置文件**：调用系统编辑器直接打开 `config.json`，方便手动调整。 | [menu.go](file:///v:/git_data/github-hosts/scripts/menu.go) `openConfigFile()` |
| 系统工具 | 11 | **系统诊断**：汇总权限、网络、DNS、配置目录、hosts 文件内容等信息，输出一份可复制的诊断报告。 | [utils.go](file:///v:/git_data/github-hosts/scripts/utils.go) `runDiagnostics()` |
| 备份管理 | 12 | **创建新备份**：将当前系统 hosts 文件复制为带时间戳（含毫秒）的备份文件。 | [backup.go](file:///v:/git_data/github-hosts/scripts/backup.go) `createNewBackup()` |
| 备份管理 | 13 | **恢复备份**：列出所有备份并交互式选择其一，恢复后当前 hosts 会被覆盖（恢复前自动再做一次备份）。 | [backup.go](file:///v:/git_data/github-hosts/scripts/backup.go) `restoreBackupMenu()` / `restoreBackup()` |
| 备份管理 | 14 | **删除备份**：列出所有备份并交互式删除指定备份。 | [backup.go](file:///v:/git_data/github-hosts/scripts/backup.go) `deleteBackupMenu()` |
| 配置管理 | 15 | **导出配置**：将当前程序的配置（更新间隔、是否自动更新、时区等）导出为 JSON 文件。 | [config.go](file:///v:/git_data/github-hosts/scripts/config.go) `exportConfigToFile()` |
| 配置管理 | 16 | **导入配置**：从 JSON 文件读取配置并覆盖当前设置。 | [config.go](file:///v:/git_data/github-hosts/scripts/config.go) `importConfigFromFile()` |
| 配置管理 | 17 | **时区设置**：选择时区的时区信息。 | [config.go](file:///v:/git_data/github-hosts/scripts/config.go) `changeTimeZone()` |
| 程序更新 | 18 | **检查程序更新**：访问 GitHub Releases 比对 tag 与本地 `.version` 文件，发现新版本后可下载替换。 | [update.go](file:///v:/git_data/github-hosts/scripts/update.go) `runUpdateCheck()` / `performUpdate()` |
| 系统 | 19 | **打开 hosts 文件**：使用系统默认编辑器打开 `C:\Windows\System32\drivers\etc\hosts` 或 `/etc/hosts`。 | [main.go](file:///v:/git_data/github-hosts/scripts/main.go) `openHostsFile()` |
| 关于 | 20 | **🐙 访问项目主页**：在浏览器中打开 `https://github.com/aspnmy/github-hosts`。 | [main.go](file:///v:/git_data/github-hosts/scripts/main.go) `openGitHubRepo()` |
| — | 0 | **退出程序** | — |

> 首次运行后，程序会在**可执行文件所在目录**写入 <code>.version</code> 文件，内容例如 <code>v0.0.0.1_nokv</code>。该文件用于「16 检查程序更新」时与 GitHub 最新 tag 对比版本。部署新版本时，请同时更新仓库根目录的 <code>.version</code> 文件。

### 5. 版本号与自更新说明

- 仓库根目录有一个 <code>.version</code> 文件（例如 <code>v0.0.0.1_nokv</code>），它是**权威版本号来源**。
- Release 时，CI（`.github/workflows/release.yml`）会根据打 tag 的版本（如 `v0.0.0.2_nokv`）构建二进制，并将其与 <code>.version</code> 一起发布。
- Go 客户端「首次运行」时会在**可执行文件所在目录**写入一份 <code>.version</code>，之后「16 检查程序更新」会读取此文件与 GitHub 最新 tag 比较。
- Shell 脚本版（`github_hosts.sh`）不依赖版本号文件——始终从线上拉取最新 hosts。

### 6. 自行编译 Go 客户端

```bash
# 进入 Go 源码目录
cd scripts

# 本地构建
go build -o github-hosts .

# 交叉编译（release.yml 中已覆盖 7 种平台，此处示例）
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o github-hosts.linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o github-hosts.darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o github-hosts.windows-amd64.exe .
```

需要 Go 1.22+。编译后以管理员 / root 身份运行即可进入菜单。


## API 文档

本项目基于 [Hono](https://hono.dev/) 框架部署在 Cloudflare Workers 上，提供以下接口（由 [src/index.ts](file:///v:/git_data/github-hosts/src/index.ts) 实现）：

| 方法 | 路径 | 描述 | 限流 |
|------|------|------|------|
| `GET` | `/` | 返回 HTML 首页模板（位于 `public/index.html`） | — |
| `GET` | `/hosts` | 返回 **纯文本格式** 的完整 hosts 文件（含区块注释与更新时间） | — |
| `GET` | `/hosts.json` | 返回 **JSON 格式** 的 IP 与域名映射数组，形如 `[["140.82.114.4","github.com"], ...]` | — |
| `GET` | `/{domain}` | 查询 **单个域名** 的实时 IP。仅限 `domains.txt` 中预定义的域名。返回 `{ip, lastUpdated, lastChecked}` | **30 请求 / 分钟** |
| `POST` | `/reset?key=<API_KEY>` | 管理接口：强制重新拉取所有域名的 DNS 解析数据。返回 `{message, entriesCount, entries}` | **5 请求 / 分钟，且需 API Key** |

> `/reset` 接口需在 `wrangler.toml` 中配置 `API_KEY` 环境变量。未匹配的 key 返回 `401 Unauthorized`。  
> `/hosts` 返回的格式为标准 hosts 文件（含 `# github hosts` 区块头），可直接写入系统 hosts。  
> 支持的域名由仓库根目录 `domains.txt` 动态加载（默认 5 分钟内存缓存），详见「远程域名配置」。

## 常见问题

- 对高度定制化的类 Unix 系统（如绿联云 DX4600 未升级版本、嵌入式 Linux、软路由系统）：推荐使用 Release 中附带的 `github_hosts.sh` 脚本，无需编译、跨平台通用，详见「3. Shell 脚本」小节。

### 权限问题
- Windows：需要以管理员身份运行
- MacOS/Linux：需要 sudo 权限

### 更新失败
- 确保网络连接和文件权限正常
- 检查 `hosts.earth-online.org` 是否可达（可使用程序菜单的「6 测试网络连接」）

## 部署指南

本项目基于 [Hono](https://hono.dev/) 框架，部署到 Cloudflare Workers。

### 前置条件

- Node.js 18+ 与 [pnpm](https://pnpm.io/)
- Cloudflare 账号（免费计划即可）

### 安装与本地开发

```bash
pnpm install
pnpm run dev -- --remote  # 本地开发（--remote 使用 Cloudflare 远程运行时）
pnpm run deploy           # 首次部署到 Cloudflare（会提示登录）
```

[![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/aspnmy/github-hosts)

### 环境变量配置（`wrangler.toml`）

| 变量 | 说明 | 示例 |
|------|------|------|
| `DOMAINS_URL` | 运行时拉取域名列表的 URL（5 分钟内存缓存） | `https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt` |
| `API_KEY` | `/reset` 管理接口的访问密钥。**部署前必须设置，否则重置接口不可用** | 任意安全字符串 |

> 如 `DOMAINS_URL` 拉取失败，会回退到 [src/constants.ts](file:///v:/git_data/github-hosts/src/constants.ts) 中内置的 `GITHUB_URLS` 列表。

### DNS 解析原理

Worker 会对每个域名轮询两个 DNS over HTTPS 提供商：

1. **Cloudflare DNS**（`1.1.1.1`）
2. **Google DNS**（`dns.google`）

并启用 **3 次重试 + 指数退避**，返回首个合法 IPv4 地址。所有数据为 **实时拉取**，不持久化存储。


## 远程域名配置

本项目支持通过仓库根目录的 `domains.txt` **自定义域名列表**（每行一个域名，以 `#` 开头的行为注释）。

### 工作原理

1. Worker 启动时会从 `wrangler.toml` 中配置的 `DOMAINS_URL` 拉取域名列表
2. 拉取到的域名会在 **Worker 内存中缓存 5 分钟**
3. 缓存失效后自动重新拉取；若拉取失败，回退到 [src/constants.ts](file:///v:/git_data/github-hosts/src/constants.ts) 中内置的 `GITHUB_URLS` 列表

### 推荐的 `DOMAINS_URL`

```
https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt
```

### 如何添加新域名

1. 修改仓库根目录 `domains.txt`，添加域名（每行一个）
2. 推送到 `nokv` 分支（或通过 PR 合并）
3. Worker 会在下一次缓存过期（最长 5 分钟）后自动生效
4. 如需立即生效，可调用 `POST /reset?key=<API_KEY>` 接口强制刷新

### 使用公共部署（无需自己部署 Worker）

如果你不想维护自己的 Cloudflare 部署，可以直接使用公共部署 `https://hosts.earth-online.org`：

- Fork 本仓库并修改 fork 中的 `domains.txt`
- 将修改通过 PR 合并回 `aspnmy/github-hosts` 的 `nokv` 分支
- 合并后公共部署会从你的域名列表中解析

> ⚠️ 重要提醒：使用共享部署时，请确保你维护的域名不包含恶意或受限域名，否则可能导致服务或安全问题。

## 项目结构

```
.
├── src/                         # Cloudflare Worker 源码（TypeScript + Hono）
│   ├── index.ts                 # 入口，路由定义
│   ├── services/hosts.ts        # DNS 查询与 hosts 格式化
│   ├── domain-config.ts         # domains.txt 拉取与缓存
│   ├── constants.ts             # 内置域名、DNS 提供商
│   └── middleware/rate-limit.ts
├── scripts/                     # Go 客户端程序（14 个 .go 文件）
│   ├── main.go                  # 入口 / 20 项菜单（dispatchChoice）
│   ├── types.go                 # 全局常量、数据结构
│   ├── banner.go                # 启动横幅与版本号读取
│   ├── install.go               # 安装 / 更新 hosts 区块（原子写入）
│   ├── uninstall.go             # 卸载 / 清理 hosts 区块（Start/End 标记）
│   ├── config.go                # 配置管理 / 时区 / 自动更新
│   ├── cron.go                  # Windows 任务计划 / Linux cron / macOS launchd
│   ├── network.go               # 并发 DNS/HTTPS 网络连接测试
│   ├── backup.go                # hosts 备份 / 恢复 / 删除
│   ├── update.go                # GitHub Release 自更新
│   ├── log.go                   # 统一日志系统（持久化句柄 + 日期切换）
│   ├── utils.go                 # 目录权限 / 系统诊断工具
│   ├── menu.go                  # 辅助功能（打开配置目录/文件 等）
│   └── github_hosts.sh          # 通用 Bash 脚本（无需 Go 运行时）
├── public/                      # Worker 静态资源（HTML 首页）
├── .github/workflows/           # CI/CD：编译二进制 + 打 tag 发布
├── domains.txt                  # 支持的域名列表
├── wrangler.toml                # Cloudflare Worker 配置
├── .version                     # Go 客户端版本号（编译时注入）
└── README.md
```

## 修订历史

> 本节记录 Go 客户端的重要修订。每次代码修订在本 README 留下对应哈希，方便追溯。

| 日期 (YYYY-MM-DD) | Commit 哈希 | 修订说明 |
|------------------|-------------|---------|
| 2026-06-12 | `8fe010c52a3b44b2b5eb19e84e2f468910a08a03` | **feat: 新增时区支持**。新增 `changeTimeZone()`、`detectSystemTimeZoneName()`、Windows `tzutil /g` 自动检测、TimeZone 字段写入配置；新增 `timeFormatStdTZ` 等时间格式常量。 |
| 2026-06-12 | (work in progress — 本次尚未提交) | **Bug 修复与代码质量**：<br>• `install.go` 安装检测改为精确匹配 `hostsStartMarker`；<br>• `uninstall.go` `cleanHostsFile` 改为基于 Start/End 标记的精确区块删除；<br>• `main.go` `checkInstallStatus` 与 `countGitHubHosts` 合并，hosts 文件只读一次；<br>• `main.go` 菜单 switch 抽出独立 `dispatchChoice()` 函数，主循环简化；<br>• `install.go` `updateHosts` 改为**原子写入**（先写临时文件校验 → `os.Rename` 替换），避免中途失败损坏 hosts；<br>• `config.go` `loadConfig` 新增字段校验（UpdateInterval非法值回退为 60，TimeZone 空值自动补全）；<br>• `log.go` 采用持久化文件句柄 + 按日期切换，避免每次日志写入都 `Open/Close`；<br>• `backup.go` 备份文件名加毫秒后缀，防止同一秒内覆盖；<br>• `update.go` `matchReleaseAsset` 取循环变量地址改为显式 `matched := asset`；<br>• `types.go` 集中定义 `hostsStartMarker`、`hostsEndMarker`、时间格式等硬编码字符串；<br>• 全局 `fmt.Scanf` / `fmt.Scanln` 替换为 `readLine` / `promptString` / `promptInt` 统一输入；<br>• 移除 `App.logger` 无用字段（`log.go` 包级函数已承担日志输出）。 |

## 鸣谢

- [TinsFox/github-hosts](https://github.com/TinsFox/github-hosts) - 原版项目
- [GitHub520](https://github.com/521xueweihan/GitHub520)
- [![Powered by DartNode](https://dartnode.com/branding/DN-Open-Source-sm.png)](https://dartnode.com "Powered by DartNode - Free VPS for Open Source")

## 许可证

[MIT](./LICENSE)
