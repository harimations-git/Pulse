package monitor

type Stats struct {
	MonitorID             int64   `json:"monitorId"`
	TotalChecks           int64   `json:"totalChecks"`
	SuccessfulChecks      int64   `json:"successfulChecks"`
	UptimePercentage      float64 `json:"uptimePercentage"`
	AverageResponseTimeMs float64 `json:"averageResponseTimeMs"`
}
