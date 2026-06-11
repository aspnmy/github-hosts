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

## API 文档

- `GET /hosts` - 获取 hosts 文件内容
- `GET /hosts.json` - 获取 JSON 格式的数据
- `GET /{domain}` - 获取指定域名的实时 DNS 解析结果
- `POST /reset` - 重新获取所有数据（需要 API 密钥）

## 常见问题

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

- 更新域名只需修改 `domains.txt` 并推送到 `main` 分支。
- Worker 会根据 `wrangler.toml` 中的 `DOMAINS_URL` 运行时拉取该文件（并有 5 分钟的缓存），也可以通过 CI 发布后立即生效。


## 鸣谢

- [TinsFox/github-hosts](https://github.com/TinsFox/github-hosts) - 原版项目
- [GitHub520](https://github.com/521xueweihan/GitHub520)
- [![Powered by DartNode](https://dartnode.com/branding/DN-Open-Source-sm.png)](https://dartnode.com "Powered by DartNode - Free VPS for Open Source")

## 许可证

[MIT](./LICENSE)
