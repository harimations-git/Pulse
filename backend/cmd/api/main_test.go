package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

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
