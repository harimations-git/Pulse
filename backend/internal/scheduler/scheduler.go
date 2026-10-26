package scheduler

import (
	"context"
	"time"
)

type Scheduler struct {
	Interval time.Duration
	OnTick   func(context.Context)
}

func NewScheduler(interval time.Duration, onTick func(context.Context)) *Scheduler {
	return &Scheduler{
		Interval: interval,
		OnTick:   onTick,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			s.OnTick(ctx)
		}
	}
}
