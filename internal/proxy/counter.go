package proxy

import (
	"math"
	"net"
	"sync"
	"time"
)

// ConnStats holds traffic statistics for one VPS tunnel.
type ConnStats struct {
	Name     string
	Provider ConnProvider

	mu         sync.Mutex
	readBytes  int64
	writeBytes int64
	lastRead   int64
	lastWrite  int64
	lastCheck  time.Time
	readRate   float64 // bytes/sec
	writeRate  float64 // bytes/sec
}

func NewConnStats(name string, provider ConnProvider) *ConnStats {
	return &ConnStats{Name: name, Provider: provider}
}

// Add records additional bytes transferred through this tunnel.
func (s *ConnStats) Add(read, write int64) {
	s.mu.Lock()
	s.readBytes += read
	s.writeBytes += write
	s.mu.Unlock()
}

// Reset zeroes all counters and rates.
func (s *ConnStats) Reset() {
	s.mu.Lock()
	s.readBytes = 0
	s.writeBytes = 0
	s.lastRead = 0
	s.lastWrite = 0
	s.readRate = 0
	s.writeRate = 0
	s.lastCheck = time.Time{}
	s.mu.Unlock()
}

// Tick calculates rates since last tick. Call every ~2 seconds.
func (s *ConnStats) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if !s.lastCheck.IsZero() {
		elapsed := now.Sub(s.lastCheck).Seconds()
		if elapsed > 0 {
			s.readRate = float64(s.readBytes-s.lastRead) / elapsed
			s.writeRate = float64(s.writeBytes-s.lastWrite) / elapsed
		}
	}
	s.lastRead = s.readBytes
	s.lastWrite = s.writeBytes
	s.lastCheck = now
}

// RatesSnapshot is a thread-safe snapshot for API responses.
type RatesSnapshot struct {
	Name       string  `json:"name"`
	ReadBps    float64 `json:"read_bps"`
	WriteBps   float64 `json:"write_bps"`
	ReadTotal  int64   `json:"read_total"`
	WriteTotal int64   `json:"write_total"`
}

// Snapshot returns a copy of the current stats.
func (s *ConnStats) Snapshot() RatesSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return RatesSnapshot{
		Name:       s.Name,
		ReadBps:    math.Round(s.readRate),
		WriteBps:   math.Round(s.writeRate),
		ReadTotal:  s.readBytes,
		WriteTotal: s.writeBytes,
	}
}

// statsConn wraps a net.Conn to track bytes transferred incrementally.
// On each Read/Write it reports to ConnStats for real-time rate tracking.
type statsConn struct {
	net.Conn
	stats *ConnStats
}

// newStatsConn wraps an existing net.Conn with stats tracking.
func newStatsConn(conn net.Conn, stats *ConnStats) *statsConn {
	return &statsConn{Conn: conn, stats: stats}
}

func (c *statsConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		// bytes received from remote → client download = write
		c.stats.Add(0, int64(n))
	}
	return n, err
}

func (c *statsConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		// bytes sent to remote → client upload = read
		c.stats.Add(int64(n), 0)
	}
	return n, err
}

func (c *statsConn) Close() error {
	return c.Conn.Close()
}

func (c *statsConn) CloseWrite() error {
	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}

func (c *statsConn) CloseRead() error {
	if cr, ok := c.Conn.(closeReader); ok {
		return cr.CloseRead()
	}
	return nil
}
