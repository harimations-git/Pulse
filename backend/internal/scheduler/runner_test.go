package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func TestRunDueChecks(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	// Protect against using the wrong database
	var dbName string
	if err := db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName); err != nil {
		t.Fatal(err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := database.NewMonitorRepository(db)
	checkRepo := database.NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/runner-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"Runner Test",
		testURL,
		60,
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, err := db.Exec(
			context.Background(),
			"DELETE FROM monitors WHERE id = $1",
			created.ID,
		)
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	})

	processed := make(map[int64]bool)
	var mu sync.Mutex

	fakeCheck := func(ctx context.Context, m monitor.Monitor) error {
		mu.Lock()
		processed[m.ID] = true
		mu.Unlock()
		return nil
	}

	err = RunDueChecks(
		ctx,
		monitorRepo,
		checkRepo,
		2,
		fakeCheck,
	)

	if err != nil {
		t.Fatal("RunDueChecks failed:", err)
	}

	mu.Lock()
	wasProcessed := processed[created.ID]
	mu.Unlock()

	if !wasProcessed {
		t.Error("Expected due monitor to be processed")
	}
}
