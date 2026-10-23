package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := map[string]string{
		"status": "ok",
	}

	json.NewEncoder(w).Encode(data)
}

func main() {

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("POST /monitors/check", checkHandler)

	fmt.Println("Pulse API running on port 8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}

}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.Hostname() == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}

	result := monitor.CheckService(req.URL, client)

	response := CheckResponse{
		URL:            result.URL,
		StatusCode:     result.StatusCode,
		ResponseTimeMs: result.ResponseTime.Seconds() * 1000,
		Healthy:        result.Healthy,
	}

	if result.Err != nil {
		response.Error = result.Err.Error()
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Failed to encode response:", err)
	}
}

type CheckResponse struct {
	URL            string  `json:"url"`
	StatusCode     int     `json:"statusCode"`
	ResponseTimeMs float64 `json:"responseTimeMs"`
	Healthy        bool    `json:"healthy"`
	Error          string  `json:"error,omitempty"`
}

type CheckRequest struct {
	URL string `json:"url"`
}
