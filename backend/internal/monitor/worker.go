package monitor

import "net/http"

func Worker(
	jobs <-chan string,
	results chan<- CheckResult,
	client *http.Client,
) {
	for url := range jobs {
		results <- CheckService(url, client)
	}
}
