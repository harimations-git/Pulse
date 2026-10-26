package scheduler

import (
	"context"
	"sync"

	"github.com/harimations-git/pulse/backend/internal/monitor"
)

type CheckFunc func(
	context.Context,
	monitor.Monitor,
) error

func Dispatch(
	ctx context.Context,
	monitors []monitor.Monitor,
	workerCount int,
	check CheckFunc,
) []error {
	if workerCount <= 0 {
		workerCount = 1
	}

	jobs := make(chan monitor.Monitor)
	errorsCh := make(chan error, len(monitors))

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for m := range jobs {
				if ctx.Err() != nil {
					return
				}

				if err := check(ctx, m); err != nil {
					errorsCh <- err
				}
			}
		}()
	}

dispatchLoop:
	for _, m := range monitors {
		if ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
			break dispatchLoop

		case jobs <- m:
		}
	}

	close(jobs)
	wg.Wait()
	close(errorsCh)

	errs := []error{}

	for err := range errorsCh {
		errs = append(errs, err)
	}

	return errs
}
