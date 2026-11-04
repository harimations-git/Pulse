package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(
	r *http.Request,
) (*http.Response, error) {
	return f(r)
}

func TestHealthHandler(t *testing.T) {

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	healthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"Expected status 200, got %d",
			recorder.Code,
		)
	}

	contentType := recorder.Header().Get("Content-Type")

	if contentType != "application/json" {
		t.Errorf(
			"Expected application/json, got %s",
			contentType,
		)
	}

	expectedBody := "{\"status\":\"ok\"}\n"

	if recorder.Body.String() != expectedBody {
		t.Errorf(
			"Expected %q, got %q",
			expectedBody,
			recorder.Body.String(),
		)
	}
}

func TestCheckHandlerEmptyURL(t *testing.T) {
	body := strings.NewReader(`{"url":""}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/monitors/check",
		body,
	)

	recorder := httptest.NewRecorder()

	checkHandler(recorder, req, nil)

	if recorder.Code != 400 {
		t.Errorf(
			"Expected status 400, got %d",
			recorder.Code,
		)
	}

	if !strings.Contains(recorder.Body.String(), "URL is required") {
		t.Errorf(
			"Expected 'URL is required', got %q",
			recorder.Body.String(),
		)
	}
}

func TestCheckHandlerInvalidJSON(t *testing.T) {

	body := strings.NewReader(`{"url":`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/monitors/check",
		body,
	)

	recorder := httptest.NewRecorder()

	checkHandler(recorder, req, nil)

	if recorder.Code != 400 {
		t.Errorf(
			"Expected status 400, got %d",
			recorder.Code,
		)
	}

	if !strings.Contains(recorder.Body.String(), "Invalid JSON") {
		t.Errorf(
			"Expected 'Invalid JSON', got %q",
			recorder.Body.String(),
		)
	}
}

func TestCheckHandlerSuccess(t *testing.T) {

	testURL := "https://example.com"

	// Create a fake HTTP client that returns 200
	client := &http.Client{
		Transport: roundTripFunc(
			func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}, nil
			},
		),
	}

	// Create JSON request with a valid URL
	body := fmt.Sprintf(`{"url":"%s"}`, testURL)

	request := httptest.NewRequest(
		http.MethodPost,
		"/monitors/check",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	// Execute the API handler
	checkHandler(recorder, request, client)

	// Verify HTTP response
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"Expected status 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode JSON response
	var response CheckResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	// Verify URL
	if response.URL != testURL {
		t.Errorf(
			"Expected URL %s, got %s",
			testURL,
			response.URL,
		)
	}

	// Verify status code
	if response.StatusCode != http.StatusOK {
		t.Errorf(
			"Expected status 200, got %d",
			response.StatusCode,
		)
	}

	// Verify healthy result
	if !response.Healthy {
		t.Error("Expected service to be healthy")
	}

	// Verify no error
	if response.Error != "" {
		t.Errorf("Expected no error, got %s", response.Error)
	}
}

func TestCheckRegisteredMonitorInvalidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"Non-numeric ID", "abc"},
		{"Negative ID", "-1"},
		{"Zero ID", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/monitors/"+tt.id+"/check",
				nil,
			)

			request.SetPathValue("id", tt.id)

			recorder := httptest.NewRecorder()

			checkRegisteredMonitorHandler(
				recorder,
				request,
				nil,
				nil,
				nil,
				nil,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf(
					"Expected status 400, got %d",
					recorder.Code,
				)
			}
		})
	}
}

func TestCheckRegisteredMonitorSuccess(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(db.Close)

	var dbName string
	if err := db.QueryRow(ctx, "SELECT current_database()").Scan(&dbName); err != nil {
		t.Fatal(err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	created, err := monitorRepo.Create(
		ctx,
		"API Integration Test",
		server.URL,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create test monitor:", err)
	}

	t.Cleanup(func() {
		_, err := db.Exec(
			context.Background(),
			"DELETE FROM monitors WHERE id = $1",
			created.ID,
		)
		if err != nil {
			t.Errorf("Failed to clean up monitor: %v", err)
		}
	})

	request := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/monitors/%d/check", created.ID),
		nil,
	)

	request.SetPathValue("id", strconv.FormatInt(created.ID, 10))

	recorder := httptest.NewRecorder()

	checkRegisteredMonitorHandler(
		recorder,
		request,
		monitorRepo,
		checkRepo,
		server.Client(),
		monitor.NewCheckGuard(),
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"Expected 201 Created, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var saved monitor.CheckRecord

	if err := json.Unmarshal(recorder.Body.Bytes(), &saved); err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	if saved.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", saved.StatusCode)
	}

	if !saved.Healthy {
		t.Error("Expected monitor to be healthy")
	}

	if saved.MonitorID != created.ID {
		t.Errorf("Expected monitor ID %d, got %d",
			created.ID, saved.MonitorID)
	}

	if saved.ErrorMessage != nil {
		t.Errorf("Expected no error, got %s", *saved.ErrorMessage)
	}

	records, err := checkRepo.GetByMonitorID(ctx, created.ID)

	if err != nil {
		t.Fatal("Failed to retrieve saved history:", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 saved check, got %d", len(records))
	}

	if records[0].ID != saved.ID {
		t.Errorf(
			"Expected saved ID %d, got %d",
			saved.ID,
			records[0].ID,
		)
	}
}

func TestAPIRejectsUnsafeURLs(t *testing.T) {
	unsafeURLs := []string{
		"http://localhost:8080",
		"http://127.0.0.1",
		"http://192.168.1.1",
		"http://169.254.169.254",
		"https://example.com:5432",
	}

	for _, targetURL := range unsafeURLs {
		t.Run(targetURL, func(t *testing.T) {

			// Test the direct URL checking endpoint
			checkBody, err := json.Marshal(CheckRequest{
				URL: targetURL,
			})
			if err != nil {
				t.Fatal(err)
			}

			checkRequest := httptest.NewRequest(
				http.MethodPost,
				"/monitors/check",
				bytes.NewReader(checkBody),
			)

			checkRecorder := httptest.NewRecorder()

			checkHandler(checkRecorder, checkRequest, nil)

			if checkRecorder.Code != http.StatusBadRequest {
				t.Errorf(
					"Expected check endpoint to return 400, got %d",
					checkRecorder.Code,
				)
			}

			// Test the monitor registration endpoint
			createBody, err := json.Marshal(CreateMonitorRequest{
				Name:            "Unsafe Monitor",
				URL:             targetURL,
				IntervalSeconds: 60,
			})
			if err != nil {
				t.Fatal(err)
			}

			createRequest := httptest.NewRequest(
				http.MethodPost,
				"/monitors",
				bytes.NewReader(createBody),
			)

			createRecorder := httptest.NewRecorder()

			createMonitorHandler(
				createRecorder,
				createRequest,
				nil,
			)

			if createRecorder.Code != http.StatusBadRequest {
				t.Errorf(
					"Expected create endpoint to return 400, got %d",
					createRecorder.Code,
				)
			}
		})
	}
}

func TestCheckRegisteredMonitorConflict(t *testing.T) {

	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	var databaseName string

	err = db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&databaseName)

	if err != nil {
		t.Fatal("Failed to verify database:", err)
	}

	if databaseName != "pulse_test" {
		t.Fatalf(
			"Expected pulse_test database, got %s",
			databaseName,
		)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://pulse-conflict-%d.example.com",
		time.Now().UnixNano(),
	)

	createdMonitor, err := monitorRepo.Create(
		ctx,
		"Conflict Test Monitor",
		testURL,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create test monitor:", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		_, err := db.Exec(
			cleanupCtx,
			"DELETE FROM monitors WHERE id = $1",
			createdMonitor.ID,
		)

		if err != nil {
			t.Errorf("Failed to clean up monitor: %v", err)
		}
	})

	guard := monitor.NewCheckGuard()

	if !guard.TryAcquire(createdMonitor.ID) {
		t.Fatal("Expected to acquire monitor")
	}
	defer guard.Release(createdMonitor.ID)

	request := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/monitors/%d/check", createdMonitor.ID),
		nil,
	)

	request.SetPathValue(
		"id",
		strconv.FormatInt(createdMonitor.ID, 10),
	)

	recorder := httptest.NewRecorder()

	checkRegisteredMonitorHandler(
		recorder,
		request,
		monitorRepo,
		checkRepo,
		nil,
		guard,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"Expected 409 Conflict, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	records, err := checkRepo.GetByMonitorID(
		ctx,
		createdMonitor.ID,
	)

	if err != nil {
		t.Fatal("Failed to retrieve check records:", err)
	}

	if len(records) != 0 {
		t.Errorf(
			"Expected no saved checks, got %d",
			len(records),
		)
	}
}

func TestGetMonitorStatsInvalidID(t *testing.T) {

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/invalid/stats",
		nil,
	)

	request.SetPathValue("id", "invalid")

	recorder := httptest.NewRecorder()

	getMonitorStatsHandler(
		recorder,
		request,
		nil,
		nil,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf(
			"Expected 400 Bad Request, got %d",
			recorder.Code,
		)
	}
}

func TestGetMonitorStatsHandler(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	// Register pool cleanup first so it runs last.
	t.Cleanup(db.Close)

	// Verify the database is safe to use for tests.
	var databaseName string
	err = db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&databaseName)

	if err != nil {
		t.Fatal(err)
	}

	if databaseName != "pulse_test" {
		t.Fatalf(
			"Expected pulse_test, got %s",
			databaseName,
		)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	t.Run("Monitor not found", func(t *testing.T) {
		// Use a valid but extremely unlikely monitor ID.
		id := "9223372036854775807"

		request := httptest.NewRequest(
			http.MethodGet,
			"/monitors/"+id+"/stats",
			nil,
		)
		request.SetPathValue("id", id)

		recorder := httptest.NewRecorder()

		getMonitorStatsHandler(
			recorder,
			request,
			monitorRepo,
			checkRepo,
		)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf(
				"Expected 404, got %d: %s",
				recorder.Code,
				recorder.Body.String(),
			)
		}
	})

	t.Run("Returns statistics", func(t *testing.T) {
		testURL := fmt.Sprintf(
			"https://example.com/stats-%d",
			time.Now().UnixNano(),
		)

		created, err := monitorRepo.Create(
			ctx,
			"Statistics API Test",
			testURL,
			60,
		)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cleanupCancel()

			_, err := db.Exec(
				cleanupCtx,
				"DELETE FROM monitors WHERE id = $1",
				created.ID,
			)

			if err != nil {
				t.Errorf("Cleanup failed: %v", err)
			}
		})

		// Save one successful and one failed check.
		checks := []monitor.CheckResult{
			{
				URL:          testURL,
				StatusCode:   200,
				ResponseTime: 100 * time.Millisecond,
				Healthy:      true,
			},
			{
				URL:          testURL,
				StatusCode:   503,
				ResponseTime: 300 * time.Millisecond,
				Healthy:      false,
			},
		}

		for _, result := range checks {
			if _, err := checkRepo.Save(
				ctx,
				created.ID,
				result,
			); err != nil {
				t.Fatal(err)
			}
		}

		request := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/monitors/%d/stats", created.ID),
			nil,
		)

		request.SetPathValue(
			"id",
			strconv.FormatInt(created.ID, 10),
		)

		recorder := httptest.NewRecorder()

		getMonitorStatsHandler(
			recorder,
			request,
			monitorRepo,
			checkRepo,
		)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"Expected 200, got %d: %s",
				recorder.Code,
				recorder.Body.String(),
			)
		}

		var stats monitor.Stats

		if err := json.NewDecoder(recorder.Body).Decode(&stats); err != nil {
			t.Fatal(err)
		}

		if stats.MonitorID != created.ID {
			t.Errorf("Incorrect monitor ID: %d", stats.MonitorID)
		}

		if stats.TotalChecks != 2 {
			t.Errorf("Expected 2 checks, got %d", stats.TotalChecks)
		}

		if stats.SuccessfulChecks != 1 {
			t.Errorf(
				"Expected 1 successful check, got %d",
				stats.SuccessfulChecks,
			)
		}

		if stats.UptimePercentage != 50 {
			t.Errorf(
				"Expected 50%% uptime, got %.2f%%",
				stats.UptimePercentage,
			)
		}

		if math.Abs(stats.AverageResponseTimeMs-200) > 0.001 {
			t.Errorf(
				"Expected 200ms average, got %.2fms",
				stats.AverageResponseTimeMs,
			)
		}
	})
}

func TestDeleteMonitorInvalidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"Non-numeric ID", "invalid"},
		{"Negative ID", "-1"},
		{"Zero ID", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodDelete,
				"/monitors/"+tt.id,
				nil,
			)

			request.SetPathValue("id", tt.id)

			recorder := httptest.NewRecorder()

			deleteMonitorHandler(
				recorder,
				request,
				nil,
				nil,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf(
					"Expected 400, got %d",
					recorder.Code,
				)
			}
		})
	}
}

func TestDeleteMonitorConflict(t *testing.T) {
	guard := monitor.NewCheckGuard()

	monitorID := int64(1)

	// Simulate an active monitoring check
	if !guard.TryAcquire(monitorID) {
		t.Fatal("Expected to acquire monitor")
	}

	defer guard.Release(monitorID)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/monitors/1",
		nil,
	)

	request.SetPathValue("id", "1")

	recorder := httptest.NewRecorder()

	deleteMonitorHandler(
		recorder,
		request,
		nil,
		guard,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"Expected 409 Conflict, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestDeleteMonitorHandlerSuccess(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	// Verify we're using the test database.
	var dbName string

	err = db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName)

	if err != nil {
		t.Fatal(err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	// Create a temporary monitor.
	testURL := fmt.Sprintf(
		"https://example.com/delete-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"Delete API Test",
		testURL,
		60,
	)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		_, err := db.Exec(
			cleanupCtx,
			"DELETE FROM monitors WHERE id = $1",
			created.ID,
		)

		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	})

	// Save a check to verify cascading deletion.
	_, err = checkRepo.Save(
		ctx,
		created.ID,
		monitor.CheckResult{
			URL:          testURL,
			StatusCode:   200,
			ResponseTime: 100 * time.Millisecond,
			Healthy:      true,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	guard := monitor.NewCheckGuard()

	// Helper to send DELETE requests.
	sendDelete := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/monitors/%d", created.ID),
			nil,
		)

		request.SetPathValue(
			"id",
			strconv.FormatInt(created.ID, 10),
		)

		recorder := httptest.NewRecorder()

		deleteMonitorHandler(
			recorder,
			request,
			monitorRepo,
			guard,
		)

		return recorder
	}

	// First deletion should succeed.
	recorder := sendDelete()

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"Expected 204, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if recorder.Body.Len() != 0 {
		t.Error("Expected an empty response body")
	}

	// Verify associated monitoring history was deleted.
	var remainingChecks int

	err = db.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM check_results WHERE monitor_id = $1",
		created.ID,
	).Scan(&remainingChecks)

	if err != nil {
		t.Fatal(err)
	}

	if remainingChecks != 0 {
		t.Errorf(
			"Expected 0 remaining checks, got %d",
			remainingChecks,
		)
	}

	// Deleting the same monitor again should return 404.
	recorder = sendDelete()

	if recorder.Code != http.StatusNotFound {
		t.Errorf(
			"Expected 404, got %d",
			recorder.Code,
		)
	}
}
