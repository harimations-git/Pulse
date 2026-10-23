package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
