package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerExecutesOnTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticked := make(chan struct{}, 1)

	s := NewScheduler(
		10*time.Millisecond,
		func(ctx context.Context) {
			select {
			case ticked <- struct{}{}:
			default:
			}
		},
	)

	done := make(chan struct{})

	go func() {
		defer close(done)
		s.Run(ctx)
	}()

	select {
	case <-ticked:
		// Success: the callback executed

	case <-time.After(time.Second):
		t.Fatal("Scheduler did not execute OnTick")
	}

	cancel()

	select {
	case <-done:
		// Scheduler exited correctly

	case <-time.After(time.Second):
		t.Fatal("Scheduler did not stop after cancellation")
	}
}

func TestSchedulerStopsWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false

	s := NewScheduler(
		time.Hour,
		func(ctx context.Context) {
			called = true
		},
	)

	s.Run(ctx)

	if called {
		t.Error("Expected OnTick not to execute")
	}
}
