package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func GetDueMonitors(
	ctx context.Context,
	monitorRepo *database.MonitorRepository,
	checkRepo *database.CheckRepository,
	now time.Time,
) ([]monitor.Monitor, error) {

	monitors, err := monitorRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve monitors: %w", err)
	}

	due := []monitor.Monitor{}

	for _, m := range monitors {
		lastChecked, err := checkRepo.GetLastChecked(ctx, m.ID)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get last check for monitor %d: %w",
				m.ID, err,
			)
		}

		interval := time.Duration(m.IntervalSeconds) * time.Second

		if IsDue(lastChecked, interval, now) {
			due = append(due, m)
		}
	}

	return due, nil
}
