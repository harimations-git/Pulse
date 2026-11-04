package monitor

import "sync"

type CheckGuard struct {
	mu     sync.Mutex
	active map[int64]struct{}
}

func NewCheckGuard() *CheckGuard {
	return &CheckGuard{
		active: make(map[int64]struct{}),
	}
}

func (g *CheckGuard) TryAcquire(monitorID int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.active[monitorID]; exists {
		return false
	}

	g.active[monitorID] = struct{}{}
	return true
}

func (g *CheckGuard) Release(monitorID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.active, monitorID)
}
