package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func TestDispatchProcessesAllMonitors(t *testing.T) {
	monitors := []monitor.Monitor{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	processed := make(map[int64]bool)
	var mu sync.Mutex

	check := func(ctx context.Context, m monitor.Monitor) error {
		mu.Lock()
		processed[m.ID] = true
		mu.Unlock()
		return nil
	}

	errs := Dispatch(
		context.Background(),
		monitors,
		2,
		check,
	)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	if len(processed) != len(monitors) {
		t.Errorf(
			"Expected %d processed monitors, got %d",
			len(monitors),
			len(processed),
		)
	}
}

func TestDispatchCollectsErrors(t *testing.T) {
	monitors := []monitor.Monitor{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	check := func(ctx context.Context, m monitor.Monitor) error {
		if m.ID == 2 {
			return errors.New("check failed")
		}

		return nil
	}

	errs := Dispatch(
		context.Background(),
		monitors,
		2,
		check,
	)

	if len(errs) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errs))
	}

	if errs[0].Error() != "check failed" {
		t.Errorf("Unexpected error: %v", errs[0])
	}
}

func TestDispatchRespectsWorkerLimit(t *testing.T) {
	monitors := []monitor.Monitor{
		{ID: 1},
		{ID: 2},
		{ID: 3},
		{ID: 4},
		{ID: 5},
	}

	var active atomic.Int32
	var maximum atomic.Int32

	check := func(ctx context.Context, m monitor.Monitor) error {
		current := active.Add(1)

		for {
			previous := maximum.Load()

			if current <= previous ||
				maximum.CompareAndSwap(previous, current) {
				break
			}
		}

		time.Sleep(20 * time.Millisecond)

		active.Add(-1)
		return nil
	}

	errs := Dispatch(
		context.Background(),
		monitors,
		2,
		check,
	)

	if len(errs) != 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	if maximum.Load() > 2 {
		t.Errorf(
			"Expected at most 2 concurrent checks, got %d",
			maximum.Load(),
		)
	}

	if maximum.Load() < 1 {
		t.Error("Expected at least one check to execute")
	}
}

func TestDispatchCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	monitors := []monitor.Monitor{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	var processed atomic.Int32

	check := func(ctx context.Context, m monitor.Monitor) error {
		processed.Add(1)
		return nil
	}

	Dispatch(ctx, monitors, 2, check)

	if processed.Load() != 0 {
		t.Errorf(
			"Expected no checks after cancellation, got %d",
			processed.Load(),
		)
	}
}
