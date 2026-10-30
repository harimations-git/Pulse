package monitor

import (
	"context"
	"net/http"
	"time"
)

func CheckService(ctx context.Context, url string, client *http.Client) CheckResult {

	start := time.Now()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return CheckResult{
			URL:          url,
			StatusCode:   0,
			ResponseTime: time.Since(start),
			Healthy:      false,
			Err:          err,
		}
	}

	res, err := client.Do(req)

	resTime := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:          url,
			StatusCode:   0,
			ResponseTime: resTime,
			Healthy:      false,
			Err:          err,
		}
	}

	defer res.Body.Close()

	isHealthy := res.StatusCode >= 200 && res.StatusCode < 300

	return CheckResult{
		URL:          url,
		StatusCode:   res.StatusCode,
		ResponseTime: resTime,
		Healthy:      isHealthy,
	}
}
