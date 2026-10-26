package scheduler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
)

func TestNewScheduledCheck(t *testing.T) {
	// 1. Create a fake website
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	// 2. Connect to the test database
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

	// Ensure we're using pulse_test
	var dbName string

	err = db.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		t.Fatal("Failed to identify database:", err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	// 3. Create repositories
	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	// 4. Register the fake website as a monitor
	created, err := monitorRepo.Create(
		ctx,
		fmt.Sprintf("Scheduled Test %d", time.Now().UnixNano()),
		server.URL,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create test monitor:", err)
	}

	// Cleanup: deleting the monitor also deletes its check results
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

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	check := NewScheduledCheck(checkRepo, client)

	err = check(ctx, created)

	if err != nil {
		t.Fatal("Scheduled check failed:", err)
	}

	records, err := checkRepo.GetByMonitorID(ctx, created.ID)

	if err != nil {
		t.Fatal("Failed to retrieve check history:", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 saved result, got %d", len(records))
	}

	saved := records[0]

	if saved.MonitorID != created.ID {
		t.Errorf("Expected monitor ID %d, got %d",
			created.ID, saved.MonitorID)
	}

	if saved.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP 200, got %d", saved.StatusCode)
	}

	if !saved.Healthy {
		t.Error("Expected service to be healthy")
	}

	if saved.ErrorMessage != nil {
		t.Errorf("Expected no error message, got %s", *saved.ErrorMessage)
	}

	if saved.CheckedAt.IsZero() {
		t.Error("Expected a populated timestamp")
	}
}
