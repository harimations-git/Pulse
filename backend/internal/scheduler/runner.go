package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
)

func RunDueChecks(
	ctx context.Context,
	monitorRepo *database.MonitorRepository,
	checkRepo *database.CheckRepository,
	workerCount int,
	check CheckFunc,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	due, err := GetDueMonitors(ctx, monitorRepo, checkRepo, time.Now())

	if err != nil {
		return fmt.Errorf("failed to find due monitors: %w", err)
	}

	errs := Dispatch(ctx, due, workerCount, check)

	return errors.Join(errs...)
}
