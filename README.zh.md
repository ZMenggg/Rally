# Rally ⚡

**Rally** — 将多台 VPS 代理连接融合为一条高速管道，聚合带宽加速下载。

单 Go 二进制，MIT 协议，Docker / 裸机均可部署。

[English](README.md) | 中文

---

## 架构

```
应用 (MeTube / qBittorrent / curl / 任何支持 SOCKS5 的软件)
    │  socks5://127.0.0.1:1080
    ▼
┌────────────────────────────────────┐
│           Rally                    │
│                                    │
│  ┌─ TCP roundrobin 转发 ────────┐  │
│  │    连接级负载均衡             │  │
│  └──────┬──────────┬────────────┘  │
│         │          │              │
│  ┌──────▼──┐  ┌────▼──────────┐   │
│  │Hysteria2│  │Hysteria2      │   │
│  │ → VPS 1 │  │ → VPS 2      │   │
│  └─────────┘  └───────────────┘   │
└────────────────────────────────────┘
```

每个新 TCP 连接轮询分发给不同 VPS。配合多路并发的下载工具（MeTube、qBittorrent、浏览器等），带宽自然叠加。

## 快速开始

### 1. 配置

```yaml
# rally.yaml
bind: ":1080"
vps:
  - name: us1
    type: hysteria2
    server: your-server.com
    port: 23872
    password: "your-password"
    down_mbps: 500
  - name: us2
    type: hysteria2
    server: your-second-server.com
    port: 44705
    password: "your-password"
```

### 2. 运行

```bash
# Docker（推荐）
docker run -d -p 1080:1080 -v ./rally.yaml:/etc/rally.yaml ghcr.io/zmenggg/rally:latest

# 裸机
rally run -c rally.yaml
```

### 3. 使用

```bash
# 任何支持 SOCKS5 的应用
curl -x socks5://127.0.0.1:1080 https://www.youtube.com/watch?v=...
```

## 带宽聚合原理

Rally 做的是 **连接级聚合**，不是数据包级聚合：

| 场景 | 是否聚合 | 说明 |
|------|---------|------|
| 同时下载 10 个文件 | ✅ **是** | 每个文件连接走不同 VPS |
| 浏览网页（几十个请求） | ✅ **是** | 请求分发到各 VPS |
| 单一大文件单连接下载 | ⚠️ **仅一路** | 一条连接只能用一台 VPS |

**最佳实践：** 配合支持多路并发的下载工具（MeTube 默认 12 路、qBittorrent、curl --parallel、yt-dlp），聚合效果最明显。

## 配置参考

```yaml
bind: ":1080"              # SOCKS5 监听地址
balance: roundrobin        # 负载均衡策略

log:
  level: info              # debug | info | warn | error

vps:
  - name: us1              # 任意标识名
    type: hysteria2        # 协议类型
    server: 192.0.2.1      # VPS 地址
    port: 23872            # Hysteria2 端口
    password: "secret"     # 认证密码
    sni: ""                # TLS SNI（默认使用 server）
    down_mbps: 500         # 最大下行带宽 (Mbps)
    up_mbps: 50            # 最大上行带宽 (Mbps)
```

## CLI 命令

```bash
rally run [--config rally.yaml]     启动代理
rally check [--config rally.yaml]   校验配置
rally list [--config rally.yaml]    列出后端
rally version                       版本信息
```

## 与旧方案对比

| 特性 | sing-box + HAProxy | Rally |
|------|-------------------|-------|
| 组件数 | 2 容器 (GPLv3 + C) | 1 二进制 (MIT) |
| 配置格式 | JSON + CFG | YAML |
| 热加载 | ❌ | ⏳ 规划中 |
| 可移植性 | 仅 Docker | Docker + 裸机 |
| 健康检查 | TCP 仅 | ⏳ 规划中 |
| 统计面板 | ❌ | ⏳ 规划中 |

## 开发

```bash
# 依赖: Go 1.24+
git clone https://github.com/ZMenggg/Rally.git
cd Rally
go build -o rally ./cmd/rally/
./rally run -c rally.yaml
```

## License

MIT
