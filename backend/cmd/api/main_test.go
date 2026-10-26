package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

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

	checkHandler(recorder, req)

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

	checkHandler(recorder, req)

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

	// Create a mock server that returns HTTP 200
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	// Create JSON request containing the mock server URL
	body := fmt.Sprintf(`{"url":"%s"}`, server.URL)

	req := httptest.NewRequest(
		http.MethodPost,
		"/monitors/check",
		strings.NewReader(body),
	)

	// Capture the API response
	recorder := httptest.NewRecorder()

	// Call our monitoring API handler
	checkHandler(recorder, req)

	// Verify HTTP status
	if recorder.Code != http.StatusOK {
		t.Errorf(
			"Expected status 200, got %d",
			recorder.Code,
		)
	}

	// Decode the JSON response
	var response CheckResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	if response.URL != server.URL {
		t.Errorf(
			"Expected URL %s, got %s",
			server.URL,
			response.URL,
		)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf(
			"Expected status 200, got %d",
			response.StatusCode,
		)
	}

	if !response.Healthy {
		t.Error("Expected service to be healthy")
	}

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
