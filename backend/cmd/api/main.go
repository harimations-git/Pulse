package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
	"github.com/harimations-git/pulse/backend/internal/scheduler"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := map[string]string{
		"status": "ok",
	}

	json.NewEncoder(w).Encode(data)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, os.Getenv("DATABASE_URL"))

	if err != nil {
		fmt.Println("Database connection failed:", err)
		return
	}
	defer db.Close()

	repo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	client := monitor.NewSafeHTTPClient()

	realCheck := scheduler.NewScheduledCheck(
		checkRepo,
		client,
	)

	onTick := func(ctx context.Context) {
		err := scheduler.RunDueChecks(
			ctx,
			repo,
			checkRepo,
			2,
			realCheck,
		)

		if err != nil {
			fmt.Println("Scheduler error:", err)
		}
	}

	fmt.Println("Connected to PostgreSQL successfully!")

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("POST /monitors/check",
		func(w http.ResponseWriter, r *http.Request) {
			checkHandler(w, r, client)
		},
	)
	http.HandleFunc("GET /monitors/{id}/results",
		func(w http.ResponseWriter, r *http.Request) {
			getCheckResultsHandler(w, r, checkRepo)
		},
	)

	http.HandleFunc("POST /monitors/{id}/check",
		func(w http.ResponseWriter, r *http.Request) {
			checkRegisteredMonitorHandler(
				w,
				r,
				repo,
				checkRepo,
				client,
			)
		},
	)

	http.HandleFunc("GET /monitors", func(w http.ResponseWriter, r *http.Request) {
		getMonitorsHandler(w, r, repo)
	})

	http.HandleFunc("POST /monitors", func(w http.ResponseWriter, r *http.Request) {
		createMonitorHandler(w, r, repo)
	})

	// Create a context that cancels when Pulse receives
	// Ctrl+C or a termination signal.
	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Start scheduler
	s := scheduler.NewScheduler(
		10*time.Second,
		onTick,
	)

	schedulerDone := make(chan struct{})

	go func() {
		defer close(schedulerDone)
		s.Run(appCtx)
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: http.DefaultServeMux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	fmt.Println("Pulse API running on port 8080")

	// Wait for a shutdown signal or server failure
	select {
	case <-appCtx.Done():
		fmt.Println("Shutdown signal received")

	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server error:", err)
		}
	}

	// Cancel scheduled monitoring
	stop()

	// Give active HTTP requests up to 10 seconds to finish
	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Println("HTTP shutdown error:", err)
	}

	// Wait until the scheduler has exited
	<-schedulerDone

	fmt.Println("Pulse shutdown complete")
}

func checkHandler(w http.ResponseWriter, r *http.Request, client *http.Client) {
	var req CheckRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if err := monitor.ValidateTargetURL(req.URL); err != nil {
		http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	result := monitor.CheckService(r.Context(), req.URL, client)

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

func getMonitorsHandler(
	w http.ResponseWriter,
	r *http.Request,
	repo *database.MonitorRepository,
) {
	// Retrieve monitors from PostgreSQL
	monitors, err := repo.GetAll(r.Context())

	// Handle database errors
	if err != nil {
		http.Error(
			w,
			"Failed to retrieve monitors",
			http.StatusInternalServerError,
		)
		return
	}

	// Return monitors as JSON
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(monitors); err != nil {
		fmt.Println("Failed to encode monitors:", err)
	}
}

func createMonitorHandler(
	w http.ResponseWriter,
	r *http.Request,
	repo *database.MonitorRepository,
) {
	var req CreateMonitorRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate monitor name
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Monitor name is required", http.StatusBadRequest)
		return
	}

	if err := monitor.ValidateTargetURL(req.URL); err != nil {
		http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.IntervalSeconds <= 0 {
		http.Error(w, "Interval must be greater than zero", http.StatusBadRequest)
		return
	}

	created, err := repo.Create(
		r.Context(),
		req.Name,
		req.URL,
		req.IntervalSeconds,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Monitor URL already exists", http.StatusConflict)
			return
		}

		http.Error(w, "Failed to create monitor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(created); err != nil {
		fmt.Println("Failed to encode monitor:", err)
	}
}

func getCheckResultsHandler(w http.ResponseWriter, r *http.Request, repo *database.CheckRepository) {
	idString := r.PathValue("id")

	monitorID, err := strconv.ParseInt(idString, 10, 64)

	if err != nil || monitorID <= 0 {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	records, err := repo.GetByMonitorID(r.Context(), monitorID)

	if err != nil {
		http.Error(w, "Failed to retrieve monitoring history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(records); err != nil {
		fmt.Println("Failed to encode check results:", err)
	}

}

func checkRegisteredMonitorHandler(
	w http.ResponseWriter,
	r *http.Request,
	monitorRepo *database.MonitorRepository,
	checkRepo *database.CheckRepository,
	client *http.Client,
) {
	idString := r.PathValue("id")

	monitorID, err := strconv.ParseInt(idString, 10, 64)

	if err != nil || monitorID <= 0 {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	registeredMonitor, err := monitorRepo.GetByID(r.Context(), monitorID)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Monitor not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Failed to retrieve monitor", http.StatusInternalServerError)
		return
	}

	result := monitor.CheckService(r.Context(), registeredMonitor.URL, client)

	saved, err := checkRepo.Save(r.Context(), registeredMonitor.ID, result)

	if err != nil {
		http.Error(w, "Failed to save check result", http.StatusInternalServerError)
		return
	}

	// 6. Return saved record as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(saved); err != nil {
		fmt.Println("Failed to encode saved check result:", err)
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

type CreateMonitorRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"intervalSeconds"`
}
