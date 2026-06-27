# GitHub Hosts 客户端工具

从墙外服务拉取域名 IP 数据，更新本地 hosts 文件。

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

修改 `source` 字段切换数据源。

## 编译

```bash
go build -o github-hosts .
```

## 使用

```bash
sudo ./github-hosts
```
