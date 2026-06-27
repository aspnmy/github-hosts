# GitHub Hosts 客户端工具 (NoKV)

从墙外 DNS 服务拉取域名 IP 映射，自动更新本地 `/etc/hosts` 文件。

支持多个数据源，通过配置文件切换：

| 数据源 | 地址 | 说明 |
|--------|------|------|
| CF Worker | `hosts.earth-online.org` | Cloudflare Workers 实时 DNS 查询 |
| GitHub Pages | `myhosts.earth-online.org` | GitHub Actions 定时解析 |
| 跳板机容器 | `127.0.0.1:18763` | 跳板机 Docker 容器批量解析 |

## 配置

编辑 `~/.aspnmy/hostsource.json`:

```json
{
  "source": "container",
  "sources": {
    "cf-worker": {
      "name": "Cloudflare Worker",
      "hosts_url": "https://hosts.earth-online.org/hosts",
      "enabled": true
    },
    "gitpage": {
      "name": "GitHub Pages",
      "hosts_url": "https://myhosts.earth-online.org/hosts.txt",
      "enabled": true
    },
    "container": {
      "name": "Jump Server",
      "hosts_url": "http://127.0.0.1:18763/hosts.txt",
      "enabled": true
    }
  }
}
```

修改 `source` 字段切换数据源，工具启动时自动读取。

## 功能

- **安装/更新** — 从数据源拉取最新 DNS 数据，写入 `/etc/hosts`
- **自动更新** — 设置定时任务（cron/launchd/计划任务），周期刷新
- **连接测试** — 验证 hosts 中各域名可达性，显示响应时间
- **系统诊断** — 检查目录权限、网络连接、配置完整性
- **备份/恢复** — 修改 hosts 前自动备份

## 编译

需要 Go 1.19+：

```bash
go build -ldflags='-s -w' -o github-hosts .
```

跨平台编译：

```bash
GOOS=linux   GOARCH=amd64 go build -o github-hosts-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o github-hosts-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o github-hosts-windows-amd64.exe .
```

## 使用

```bash
sudo ./github-hosts
```

首次运行选择 `1. 安装/更新`，按向导完成配置。之后可开启自动更新，定时从墙外刷新 DNS 数据。
