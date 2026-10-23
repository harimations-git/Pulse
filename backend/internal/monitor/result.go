package monitor

import "time"

type CheckResult struct {
	URL          string
	StatusCode   int
	ResponseTime time.Duration
	Healthy      bool
	Err          error
}
