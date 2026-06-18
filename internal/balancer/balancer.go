package balancer

import (
	"sync/atomic"

	"github.com/ZMenggg/Rally-go/internal/proxy"
)

// Backend wraps a proxy.ConnProvider with load-balancing metadata.
type Backend struct {
	Name     string
	Provider proxy.ConnProvider
	Weight   int

	active  int64
	healthy int32 // 0=healthy (default), 1=unhealthy
}

// IsHealthy reports whether the backend is currently considered healthy.
func (b *Backend) IsHealthy() bool {
	return atomic.LoadInt32(&b.healthy) == 0
}

// SetHealthy marks the backend as healthy or unhealthy.
func (b *Backend) SetHealthy(h bool) {
	v := int32(0)
	if !h {
		v = 1
	}
	atomic.StoreInt32(&b.healthy, v)
}

// SetHealth updates the healthy flag for a named backend.
func (b *Balancer) SetHealth(name string, healthy bool) {
	for _, be := range b.backends {
		if be.Name == name {
			be.SetHealthy(healthy)
			return
		}
	}
}

// BackendInfo is returned for status reporting.
type BackendInfo struct {
	Name      string `json:"name"`
	Active    int64  `json:"active"`
	Connected bool   `json:"connected"`
}

// Balancer distributes connections across backends.
type Balancer struct {
	backends []*Backend
	strategy string
	counter  atomic.Uint64
}

// NewWithStrategy creates a Balancer with a specific strategy.
func NewWithStrategy(backends []*Backend, strategy string) *Balancer {
	switch strategy {
	case "leastconn", "roundrobin":
		// supported
	default:
		strategy = "roundrobin"
	}
	return &Balancer{
		backends: backends,
		strategy: strategy,
	}
}

// Next picks the next backend based on the balancing strategy.
func (b *Balancer) Next() *Backend {
	if len(b.backends) == 0 {
		return nil
	}

	switch b.strategy {
	case "leastconn":
		return b.leastConn()
	default:
		return b.roundRobin()
	}
}

func (b *Balancer) roundRobin() *Backend {
	n := len(b.backends)
	if n == 0 {
		return nil
	}
	start := int(b.counter.Add(1)-1) % n
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		if b.backends[idx].IsHealthy() {
			return b.backends[idx]
		}
	}
	// Fallback: all unhealthy
	return b.backends[start]
}

func (b *Balancer) leastConn() *Backend {
	n := len(b.backends)
	if n == 0 {
		return nil
	}
	var best *Backend
	var bestActive int64 = 1<<63 - 1
	for _, be := range b.backends {
		if !be.IsHealthy() {
			continue
		}
		a := atomic.LoadInt64(&be.active)
		if a < bestActive {
			best = be
			bestActive = a
		}
	}
	// Fallback: all unhealthy, use first
	if best == nil {
		return b.backends[0]
	}
	return best
}

// Acquire marks a connection as active on a backend.
func (b *Balancer) Acquire(be *Backend) {
	atomic.AddInt64(&be.active, 1)
}

// Release marks a connection as released from a backend.
func (b *Balancer) Release(be *Backend) {
	atomic.AddInt64(&be.active, -1)
}

// Backends returns the current backend list.
func (b *Balancer) Backends() []*Backend {
	return b.backends
}

// Info returns a snapshot of all backends.
func (b *Balancer) Info() []BackendInfo {
	info := make([]BackendInfo, len(b.backends))
	for i, be := range b.backends {
		info[i] = BackendInfo{
			Name:      be.Name,
			Active:    atomic.LoadInt64(&be.active),
			Connected: be.IsHealthy(),
		}
	}
	return info
}
