package monitor

import "time"

type CheckRecord struct {
	ID             int64     `json:"id"`
	MonitorID      int64     `json:"monitorId"`
	StatusCode     int       `json:"statusCode"`
	ResponseTimeMs float64   `json:"responseTimeMs"`
	Healthy        bool      `json:"healthy"`
	ErrorMessage   *string   `json:"errorMessage"`
	CheckedAt      time.Time `json:"checkedAt"`
}
