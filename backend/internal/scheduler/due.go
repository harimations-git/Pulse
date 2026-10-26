package scheduler

import "time"

func IsDue(
	lastChecked *time.Time,
	interval time.Duration,
	now time.Time,
) bool {
	// Never checked before
	if lastChecked == nil {
		return true
	}

	nextCheck := lastChecked.Add(interval)

	return !now.Before(nextCheck)
}
