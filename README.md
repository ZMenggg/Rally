# Rally ⚡

**Rally** — 将多台 VPS 代理连接融合为一条高速管道，聚合带宽加速下载。

Rally fuses multiple VPS proxy connections into a single SOCKS5 endpoint,
aggregating their bandwidth for faster downloads.

English | [中文](README.zh.md)

---

## Architecture

```
App (MeTube / qBittorrent / curl / any SOCKS5 app)
    │  socks5://127.0.0.1:1080
    ▼
┌─────────────────────────────────────┐
│         Rally                       │
│                                     │
│  ┌─ TCP Forwarder (roundrobin) ──┐  │
│  │   connection-level balancing  │  │
│  └──────────┬──────────┬─────────┘  │
│             │          │            │
│    ┌────────▼──┐ ┌─────▼────────┐  │
│    │ Hysteria2 │ │ Hysteria2    │  │
│    │  → VPS 1  │ │  → VPS 2    │  │
│    └───────────┘ └──────────────┘  │
└─────────────────────────────────────┘
```

Each new TCP connection is routed to a different VPS in round-robin order.
With multiple concurrent connections (typical in download managers, browsers,
and video tools), bandwidth is naturally aggregated across all VPS backends.

## Quick Start

### 1. Configuration

```yaml
# rally.yaml
bind: ":1080"
balance: roundrobin

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

### 2. Run

#### Docker (recommended)

```bash
docker run -d \
  -p 1080:1080 \
  -v ./rally.yaml:/etc/rally.yaml \
  ghcr.io/zmenggg/rally:latest
```

#### Native binary

```bash
rally run -c rally.yaml
```

### 3. Use

```bash
# Any app that supports SOCKS5 proxy
curl -x socks5://127.0.0.1:1080 https://www.youtube.com/watch?v=...
```

## How Bandwidth Aggregation Works

Rally does **connection-level aggregation**, not packet-level.

| Scenario | Works? | Explanation |
|----------|--------|-------------|
| Download 10 files simultaneously | ✅ **Yes** | Each file connection goes to a different VPS |
| Browse a webpage (dozens of requests) | ✅ **Yes** | Each request distributed across VPSes |
| Single TCP stream (e.g., one large file) | ⚠️ **One VPS only** | A single connection can only use one VPS |

**For maximum aggregation**, use a downloader that opens multiple connections
(most modern tools do: yt-dlp, curl --parallel, MeTube, qBittorrent, etc.).

## Configuration Reference

### `rally.yaml`

```yaml
bind: ":1080"              # SOCKS5 listen address (default: :1080)
balance: roundrobin        # Load balancing strategy (roundrobin only for now)

log:
  level: info              # debug | info | warn | error

vps:
  - name: us1              # Any name for identification
    type: hysteria2        # Protocol (hysteria2 only for now; shadowsocks, vless planned)
    server: 192.0.2.1      # VPS server address
    port: 23872            # VPS Hysteria2 port
    password: "secret"     # Auth password
    sni: ""                # TLS SNI (defaults to server)
    down_mbps: 500         # Max download bandwidth (Mbps)
    up_mbps: 50            # Max upload bandwidth (Mbps)
```

## CLI Reference

```bash
rally run [--config rally.yaml]     Start the proxy server
rally check [--config rally.yaml]   Validate configuration
rally list [--config rally.yaml]    List backends
rally version                       Print version
```

## Comparison

| Feature | sing-box + HAProxy | Rally |
|---------|-------------------|-------|
| Components | 2 containers (sing-box GPLv3 + HAProxy) | 1 binary (MIT) |
| Config format | JSON + CFG | YAML |
| Hot reload | ❌ | ⏳ Planned |
| Portability | Docker only | Docker + native |
| Health check | TCP only | ⏳ Planned |
| Stats dashboard | ❌ | ⏳ Planned |

## Development

```bash
# Prerequisites: Go 1.24+
git clone https://github.com/ZMenggg/Rally.git
cd Rally

# Build
go build -o rally ./cmd/rally/

# Run with config
./rally run -c rally.yaml

# Test config
./rally check -c rally.yaml
```

### Project Structure

```
Rally/
├── cmd/rally/main.go        # Entry point
├── internal/
│   ├── config/config.go     # YAML config parser
│   ├── balancer/balancer.go # Round-robin load balancer
│   ├── proxy/forward.go     # TCP forwarder (replaces HAProxy)
│   ├── runner/runner.go     # Lifecycle & Hysteria2 client management
│   └── cli/app.go           # CLI command handling
├── Dockerfile               # Multi-stage build
└── rally.yaml               # Example config
```

## License

MIT
