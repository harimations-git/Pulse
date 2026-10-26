package scheduler

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func TestGetDueMonitors(t *testing.T) {
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

	err = db.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		t.Fatal("Failed to identify database:", err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	uniqueID := time.Now().UnixNano()

	urlA := fmt.Sprintf(
		"https://example.com/due-%d",
		uniqueID,
	)

	urlB := fmt.Sprintf(
		"https://example.com/not-due-%d",
		uniqueID,
	)

	monitorA, err := monitorRepo.Create(
		ctx,
		"Due Monitor",
		urlA,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create Monitor A:", err)
	}

	t.Cleanup(func() {
		_, err := db.Exec(
			context.Background(),
			"DELETE FROM monitors WHERE id = $1",
			monitorA.ID,
		)

		if err != nil {
			t.Errorf("Failed to clean up Monitor A: %v", err)
		}
	})

	monitorB, err := monitorRepo.Create(
		ctx,
		"Not Due Monitor",
		urlB,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create Monitor B:", err)
	}

	t.Cleanup(func() {
		_, err := db.Exec(
			context.Background(),
			"DELETE FROM monitors WHERE id = $1",
			monitorB.ID,
		)

		if err != nil {
			t.Errorf("Failed to clean up Monitor B: %v", err)
		}
	})

	result := monitor.CheckResult{
		URL:          monitorB.URL,
		StatusCode:   200,
		ResponseTime: 150 * time.Millisecond,
		Healthy:      true,
	}

	_, err = checkRepo.Save(ctx, monitorB.ID, result)
	if err != nil {
		t.Fatal("Failed to save check for Monitor B:", err)
	}

	dueMonitors, err := GetDueMonitors(
		ctx,
		monitorRepo,
		checkRepo,
		time.Now(),
	)

	if err != nil {
		t.Fatal("Failed to retrieve due monitors:", err)
	}

	foundA := false
	foundB := false

	for _, m := range dueMonitors {
		if m.ID == monitorA.ID {
			foundA = true
		}

		if m.ID == monitorB.ID {
			foundB = true
		}
	}

	if !foundA {
		t.Error("Expected Monitor A to be due")
	}

	if foundB {
		t.Error("Expected Monitor B not to be due")
	}
}
