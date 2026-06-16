package proxy

import (
	"math"
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
