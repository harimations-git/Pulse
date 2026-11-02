package monitor

import "testing"

func TestCheckGuard(t *testing.T) {
	guard := NewCheckGuard()

	if !guard.TryAcquire(1) {
		t.Fatal("Expected first check to be allowed")
	}

	if guard.TryAcquire(1) {
		t.Error("Expected duplicate check to be blocked")
	}

	if !guard.TryAcquire(2) {
		t.Error("Expected different monitor to be allowed")
	}

	guard.Release(1)

	if !guard.TryAcquire(1) {
		t.Error("Expected monitor to be available after release")
	}
}
