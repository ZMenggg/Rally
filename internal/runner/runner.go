package runner

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	hyclient "github.com/apernet/hysteria/core/v2/client"

	"github.com/ZMenggg/Rally/internal/balancer"
	"github.com/ZMenggg/Rally/internal/config"
	"github.com/ZMenggg/Rally/internal/logger"
	"github.com/ZMenggg/Rally/internal/proxy"
)

// Runner manages the lifecycle of the proxy server.
type Runner struct {
	cfg     *config.Config
	clients []hyclient.Client
	cleanup []func()

	mu       sync.Mutex
	running  bool
	cancel   chan struct{}
	balancer *balancer.Balancer
	fwd      *proxy.Forwarder
}

// New creates a new Runner.
func New(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg}
}

// Run starts the proxy server.
func (r *Runner) Run() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("already running")
	}
	r.running = true
	r.cancel = make(chan struct{})
	r.mu.Unlock()

	backends, err := r.buildBackends()
	if err != nil {
		return fmt.Errorf("build backends: %w", err)
	}
	if len(backends) == 0 {
		return fmt.Errorf("no VPS backends configured")
	}

	b := balancer.NewWithStrategy(backends, r.cfg.Balance)

	picker := func() (*proxy.Backend, func()) {
		be := b.Next()
		if be == nil {
			return nil, nil
		}
		b.Acquire(be)
		return &proxy.Backend{
			Name:         be.Name,
			ConnProvider: be.Provider,
		}, func() { b.Release(be) }
	}

	fwd := proxy.NewForwarder(picker)

	r.mu.Lock()
	r.balancer = b
	r.fwd = fwd
	r.mu.Unlock()

	logger.Info("Rally started with %d VPS backend(s), listening on %s",
		len(backends), r.cfg.Bind)

	err = fwd.Serve(r.cfg.Bind)

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
	return err
}

// ReloadConfig stops the current proxy and restarts with a new config.
func (r *Runner) ReloadConfig(cfg *config.Config) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return fmt.Errorf("not running")
	}
	if r.fwd != nil {
		r.fwd.Stop()
	}
	r.mu.Unlock()

	r.Close()

	r.mu.Lock()
	r.cfg = cfg
	r.mu.Unlock()

	logger.Info("Config reloaded, restarting with %d VPS backend(s)", len(cfg.VPS))
	return r.Run()
}

// Close shuts down all tunnel clients.
func (r *Runner) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.clients {
		if c != nil {
			c.Close()
		}
	}
	r.clients = nil

	for _, fn := range r.cleanup {
		fn()
	}
	r.cleanup = nil
}

// Balancer returns the current balancer (may be nil before first Run).
// Forwarder returns the current forwarder (may be nil before first Run).
func (r *Runner) Forwarder() *proxy.Forwarder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fwd
}

func (r *Runner) Balancer() *balancer.Balancer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.balancer
}

// IsEnabled checks if a VPS should be active.
func isEnabled(vps config.VPS) bool {
	if vps.Enabled == nil {
		return true // default enabled
	}
	return *vps.Enabled
}

// buildBackends creates a Balancer backend for each VPS.
func (r *Runner) buildBackends() ([]*balancer.Backend, error) {
	var backends []*balancer.Backend

	for _, vps := range r.cfg.VPS {
		if !isEnabled(vps) {
			logger.Debug("skipping disabled node: %s", vps.Name)
			continue
		}
		provider, err := r.startClient(vps)
		if err != nil {
			logger.Warn("failed to start client for %s: %v", vps.Name, err)
			continue
		}
		backends = append(backends, &balancer.Backend{
			Name:     vps.Name,
			Provider: provider,
			Weight:   1,
		})
	}

	return backends, nil
}

// startClient starts the appropriate tunnel client for the given VPS config.
func (r *Runner) startClient(vps config.VPS) (proxy.ConnProvider, error) {
	switch vps.Type {
	case "hysteria2", "":
		return r.startHysteria2(vps)
	case "socks5":
		return r.startSocks5(vps), nil
	case "ss", "shadowsocks":
		return r.startShadowsocks(vps)
	case "trojan":
		return r.startTrojan(vps), nil
	case "vless":
		return r.startVLESS(vps)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", vps.Type)
	}
}

// ─── Hysteria2 ──────────────────────────────────────────────────────────────

func (r *Runner) startHysteria2(vps config.VPS) (proxy.ConnProvider, error) {
	logger.Info("starting Hysteria2 client for %s (%s:%d)", vps.Name, vps.Server, vps.Port)

	serverAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(vps.Server, strconv.Itoa(vps.Port)))
	if err != nil {
		return nil, fmt.Errorf("resolve server addr: %w", err)
	}

	tlsServerName := vps.SNI
	if tlsServerName == "" {
		tlsServerName = vps.Server
	}

	cfg := &hyclient.Config{
		ServerAddr: serverAddr,
		Auth:       vps.Password,
		TLSConfig: hyclient.TLSConfig{
			ServerName:        tlsServerName,
			InsecureSkipVerify: false,
		},
		BandwidthConfig: hyclient.BandwidthConfig{
			MaxTx: uint64(vps.UpMbps) * 1_000_000 / 8,
			MaxRx: uint64(vps.DownMbps) * 1_000_000 / 8,
		},
	}

	client, _, err := hyclient.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("hysteria2 new client: %w", err)
	}

	r.mu.Lock()
	r.clients = append(r.clients, client)
	r.mu.Unlock()

	logger.Info("Hysteria2 client for %s connected", vps.Name)
	return &hyProvider{name: vps.Name, client: client}, nil
}

type hyProvider struct {
	name   string
	client hyclient.Client
}

func (p *hyProvider) Name() string { return p.name }
func (p *hyProvider) Dial(addr string) (net.Conn, error) {
	return p.client.TCP(addr)
}

// ─── SOCKS5 ─────────────────────────────────────────────────────────────────

func (r *Runner) startSocks5(vps config.VPS) proxy.ConnProvider {
	server := net.JoinHostPort(vps.Server, strconv.Itoa(vps.Port))
	logger.Info("starting SOCKS5 client for %s (%s)", vps.Name, server)
	return proxy.NewSOCKS5Provider(vps.Name, server)
}

// ─── Shadowsocks ────────────────────────────────────────────────────────────

func (r *Runner) startShadowsocks(vps config.VPS) (proxy.ConnProvider, error) {
	server := net.JoinHostPort(vps.Server, strconv.Itoa(vps.Port))
	logger.Info("starting Shadowsocks client for %s (%s cipher=%s)",
		vps.Name, server, vps.Cipher)
	return proxy.NewShadowsocksProvider(vps.Name, server, vps.Cipher, vps.Password)
}

// ─── Trojan ─────────────────────────────────────────────────────────────────

func (r *Runner) startTrojan(vps config.VPS) proxy.ConnProvider {
	server := net.JoinHostPort(vps.Server, strconv.Itoa(vps.Port))
	logger.Info("starting Trojan client for %s (%s)", vps.Name, server)
	return proxy.NewTrojanProvider(vps.Name, server, vps.Password, vps.SNI)
}

// ─── VLESS ──────────────────────────────────────────────────────────────────

func (r *Runner) startVLESS(vps config.VPS) (proxy.ConnProvider, error) {
	server := net.JoinHostPort(vps.Server, strconv.Itoa(vps.Port))
	logger.Info("starting VLESS client for %s (%s)", vps.Name, server)
	return proxy.NewVLESSProvider(vps.Name, server, vps.UUID, vps.Flow, vps.SNI)
}
