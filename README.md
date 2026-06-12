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

- 🚀 使用 Cloudflare Workers 部署，无需服务器
- 🌍 多 DNS 服务支持（Cloudflare DNS、Google DNS）
- ⚡️ 每次请求实时 DNS 查询，获取最新 IP 记录
- 🔄 提供多种使用方式（脚本、手动、工具）
- 📡 提供 REST API 接口
- 🚫 **无需 KV**，无需担心配额耗尽

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
  <p>在管理员权限的 PowerShell 中执行：</p>
  <pre><code class="language-powershell">irm https://github.com/aspnmy/github-hosts/releases/tag/v0.0.0.1_nokv/github-hosts.windows-amd64.exe | iex</code></pre>

  <h4>Linux 用户</h4>
  <pre><code class="language-bash">sudo curl -fsSL https://github.com/aspnmy/github-hosts/releases/tag/v0.0.0.1_nokv/github-hosts.linux-amd64 -o github-hosts && sudo chmod +x ./github-hosts && ./github-hosts</code></pre>

  <p>更多版本请查看 <a href="https://github.com/aspnmy/github-hosts/releases">Release 页面</a></p>

## API 文档

- `GET /hosts` - 获取 hosts 文件内容
- `GET /hosts.json` - 获取 JSON 格式的数据
- `GET /{domain}` - 获取指定域名的实时 DNS 解析结果
- `POST /reset` - 重新获取所有数据（需要 API 密钥）

## 常见问题
- 对阉割的高度定制化linux系统的支持：scripts\github_hosts.sh 脚本专门用于一些高度定制化系统的运行，比如绿联云DX4600系列未升级的版本
### 权限问题
- Windows：需要以管理员身份运行
- MacOS/Linux：需要 sudo 权限

### 更新失败
- 确保网络连接和文件权限正常

## 部署指南

1. Fork 本项目
2. 创建 Cloudflare Workers 账号
3. 安装并部署：
```bash
pnpm install
pnpm run dev -- --remote  # 本地开发（需 --remote 使用远程运行时）
pnpm run deploy           # 部署到 Cloudflare
```

[![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/aspnmy/github-hosts)

## 远程域名配置

本项目支持将域名配置放在仓库根目录的 `domains.txt` 中（每行一个域名，支持 `#` 注释）。

- 更新域名只需修改 `domains.txt` 并推送到 `nokv` 分支。
- Worker 会根据 `wrangler.toml` 中的 `DOMAINS_URL` 运行时拉取该文件（并有 5 分钟的缓存），也可以通过 CI 发布后立即生效。

- 推荐使用 GitHub raw 地址作为运行时来源：
   `https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt`。
- 维护方式：更新该 `domains.txt` 并推送到 `nokv` 分支，Worker 会从 `wrangler.toml` 中的 `DOMAINS_URL` 拉取该文件（默认缓存 5 分钟）。如需立即生效，请在推送后触发 CI/部署。

如果你不想维护自己的 Cloudflare 部署，也可以使用我的公共部署：

- Fork 本仓库的 `nokv` 分支（或直接 fork 本项目），修改 fork 中的 `domains.txt` 后，将修改推送回我仓库的 `nokv` 分支（即直接向 `aspnmy/github-hosts` 的 `nokv` 分支提交或通过 PR 合并）。
- 推送后，我在 Cloudflare 上的 Worker 将会读取该 `domains.txt`（由 `wrangler.toml` 中的 `DOMAINS_URL` 指定的 raw 地址），从而使用我已部署的 Worker 解析你维护的域名列表。

重要提醒：使用该共享部署时，保证你维护的域名列表不包含任何位于黑名单中的域名（例如恶意域名或受限域名），否则可能导致服务或安全问题。


## 鸣谢

- [TinsFox/github-hosts](https://github.com/TinsFox/github-hosts) - 原版项目
- [GitHub520](https://github.com/521xueweihan/GitHub520)
- [![Powered by DartNode](https://dartnode.com/branding/DN-Open-Source-sm.png)](https://dartnode.com "Powered by DartNode - Free VPS for Open Source")

## 许可证

[MIT](./LICENSE)
