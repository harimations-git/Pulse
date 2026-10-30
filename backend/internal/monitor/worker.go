package monitor

import (
	"context"
	"net/http"
)

func Worker(
	jobs <-chan string,
	results chan<- CheckResult,
	client *http.Client,
	ctx context.Context,
) {
	for url := range jobs {
		results <- CheckService(ctx, url, client)
	}
}
