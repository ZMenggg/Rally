package balancer

import (
	"sync/atomic"

	"github.com/ZMenggg/Rally/internal/proxy"
)

// Backend wraps a proxy.ConnProvider with load-balancing metadata.
type Backend struct {
	Name     string
	Provider proxy.ConnProvider
	Weight   int

	active int64
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
	n := b.counter.Add(1) - 1
	return b.backends[int(n)%len(b.backends)]
}

func (b *Balancer) leastConn() *Backend {
	best := b.backends[0]
	bestActive := atomic.LoadInt64(&best.active)
	for _, be := range b.backends[1:] {
		a := atomic.LoadInt64(&be.active)
		if a < bestActive {
			best = be
			bestActive = a
		}
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
			Connected: true,
		}
	}
	return info
}
