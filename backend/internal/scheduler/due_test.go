package scheduler

import (
	"testing"
	"time"
)

func TestIsDue(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		lastChecked *time.Time
		interval    time.Duration
		expected    bool
	}{
		{
			name:        "Never checked",
			lastChecked: nil,
			interval:    time.Minute,
			expected:    true,
		},
		{
			name:        "Overdue",
			lastChecked: timePointer(now.Add(-90 * time.Second)),
			interval:    time.Minute,
			expected:    true,
		},
		{
			name:        "Not yet due",
			lastChecked: timePointer(now.Add(-30 * time.Second)),
			interval:    time.Minute,
			expected:    false,
		},
		{
			name:        "Exactly due",
			lastChecked: timePointer(now.Add(-time.Minute)),
			interval:    time.Minute,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsDue(tt.lastChecked, tt.interval, now)

			if actual != tt.expected {
				t.Errorf(
					"Expected %t, got %t",
					tt.expected,
					actual,
				)
			}
		})
	}
}

func timePointer(t time.Time) *time.Time {
	return &t
}
