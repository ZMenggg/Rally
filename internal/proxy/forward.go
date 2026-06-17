package proxy

import (
	"net"
	"sync"
	"time"

	"github.com/ZMenggg/Rally/internal/logger"
)

// ConnProvider provides a pre-established connection to a backend.
type ConnProvider interface {
	Dial(addr string) (net.Conn, error)
	Name() string
}

// Backend wraps a ConnProvider with its display name.
type Backend struct {
	Name string
	ConnProvider
}

// BackendPicker returns the next Backend to use and a release function.
type BackendPicker func() (*Backend, func())

// Forwarder handles proxying TCP connections.
type Forwarder struct {
	pick     BackendPicker
	listener net.Listener
	stats    map[string]*ConnStats
	statsMu  sync.RWMutex
	tickDone chan struct{}
	stopOnce sync.Once
}

// NewForwarder creates a Forwarder.
func NewForwarder(pick BackendPicker) *Forwarder {
	return &Forwarder{
		pick:     pick,
		stats:    make(map[string]*ConnStats),
		tickDone: make(chan struct{}),
	}
}

// GetOrCreateStats returns the ConnStats for a given provider name.
func (f *Forwarder) GetOrCreateStats(name string, provider ConnProvider) *ConnStats {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()
	if s, ok := f.stats[name]; ok {
		return s
	}
	s := NewConnStats(name, provider)
	f.stats[name] = s
	return s
}

// AllStats returns a snapshot of all stats.
func (f *Forwarder) AllStats() []RatesSnapshot {
	f.statsMu.RLock()
	names := make([]string, 0, len(f.stats))
	for name := range f.stats {
		names = append(names, name)
	}
	f.statsMu.RUnlock()

	result := make([]RatesSnapshot, 0, len(names))
	for _, name := range names {
		f.statsMu.RLock()
		s := f.stats[name]
		f.statsMu.RUnlock()
		if s != nil {
			result = append(result, s.Snapshot())
		}
	}
	return result
}

// ResetStats zeroes all traffic counters.
func (f *Forwarder) ResetStats() {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()
	for _, s := range f.stats {
		s.Reset()
	}
}

// tickAll calls Tick on all stats.
func (f *Forwarder) tickAll() {
	f.statsMu.RLock()
	stats := make([]*ConnStats, 0, len(f.stats))
	for _, s := range f.stats {
		stats = append(stats, s)
	}
	f.statsMu.RUnlock()
	for _, s := range stats {
		s.Tick()
	}
}

// Serve starts a TCP listener.
func (f *Forwarder) Serve(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	f.listener = listener

	logger.Info("SOCKS5 proxy listening on %s", addr)

	// Start ticker for rate calculation
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				f.tickAll()
			case <-f.tickDone:
				return
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				logger.Warn("accept error: %v", err)
				continue
			}
			// Check if listener was closed by Stop() (SIGHUP reload)
			select {
			case <-f.tickDone:
				return nil // graceful shutdown, not an error
			default:
				return err
			}
		}
		go f.handleConn(conn)
	}
}

// Stop closes the listener.
func (f *Forwarder) Stop() {
	f.stopOnce.Do(func() {
		close(f.tickDone)
		if f.listener != nil {
			f.listener.Close()
		}
	})
}

func (f *Forwarder) handleConn(client net.Conn) {
	defer client.Close()

	backend, release := f.pick()
	if backend != nil {
		logger.Debug("PICKED backend=%s", backend.Name)
	} else {
		logger.Debug("PICKED no backend available")
	}
	if backend == nil {
		logger.Warn("no backends available")
		return
	}
	if release != nil {
		defer release()
	}

	// Get or create stats tracker for this backend
	stats := f.GetOrCreateStats(backend.Name, backend.ConnProvider)

	sp := &SOCKS5Proxy{
		Dial: func(addr string) (net.Conn, error) {
			raw, err := backend.Dial(addr)
			if err != nil {
				return nil, err
			}
			// Wrap the tunnel connection with real-time counting
			return newStatsConn(raw, stats), nil
		},
	}
	sp.Handle(client)
}
