package monitor

import (
	"fmt"
	"net/http"
	"time"
)

func CheckService(url string, client *http.Client) CheckResult {

	start := time.Now()
	res, err := client.Get(url)
	resTime := time.Since(start)

	if err != nil {
		fmt.Println("Something went wrong: ", err)
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
