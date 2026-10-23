package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func main() {
	urls := []string{
		"https://harimations-git.github.io/",
		"https://google.com",
		"https://github.com",
		"https://www.youtube.com/watch?v=dDr0gTqMkyE",
		"https://www.linkedin.com/notifications/",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	jobs := make(chan string)
	results := make(chan monitor.CheckResult)

	workerCount := 2

	start := time.Now()

	// Start the workers
	for range workerCount {
		go monitor.Worker(jobs, results, client)
	}

	fmt.Println("Pulse - Service Monitor")

	// Submit URLs to the workers
	go func() {
		for _, url := range urls {
			jobs <- url
		}

		close(jobs)
	}()

	// Collect the results
	checkResults := make([]monitor.CheckResult, 0, len(urls))

	for range urls {
		result := <-results
		checkResults = append(checkResults, result)
	}

	elapsedTime := time.Since(start)

	// Display the results
	for _, result := range checkResults {
		fmt.Println("URL:", result.URL)
		fmt.Println("HTTP Status:", result.StatusCode)
		fmt.Println("Response Time:", result.ResponseTime)
		fmt.Println("Healthy:", result.Healthy)
		fmt.Println()
	}

	fmt.Println("Total elapsed time:", elapsedTime)
}
