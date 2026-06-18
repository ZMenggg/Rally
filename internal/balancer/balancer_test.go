package balancer

import "testing"

func TestInfoReflectsHealth(t *testing.T) {
	be := &Backend{Name: "node-1"}
	b := NewWithStrategy([]*Backend{be}, "roundrobin")
	be.SetHealthy(false)

	info := b.Info()
	if len(info) != 1 {
		t.Fatalf("len(info) = %d, want 1", len(info))
	}
	if info[0].Connected {
		t.Fatal("Info() reported unhealthy backend as connected")
	}
}
