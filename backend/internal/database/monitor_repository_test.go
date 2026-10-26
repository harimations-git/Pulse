package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMonitorRepositoryCreate(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	// Connect to the test database
	ctx := context.Background()

	db, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(db.Close)

	// Create repository
	repo := NewMonitorRepository(db)

	//monitor with a unique URL
	testURL := fmt.Sprintf(
		"https://example.com/test-%d",
		time.Now().UnixNano(),
	)

	created, err := repo.Create(
		ctx,
		"Test Monitor",
		testURL,
		60,
	)

	if err != nil {
		t.Fatal("Failed to create monitor:", err)
	}

	// Verify PostgreSQL generated an ID
	if created.ID <= 0 {
		t.Errorf("Expected positive ID, got %d", created.ID)
	}

	// Verify correct values were stored
	if created.Name != "Test Monitor" {
		t.Errorf("Expected Test Monitor, got %s", created.Name)
	}

	if created.URL != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, created.URL)
	}

	// Remove the test record after completion
	t.Cleanup(func() {
		_, err := db.Exec(
			context.Background(),
			"DELETE FROM monitors WHERE id = $1",
			created.ID,
		)
		if err != nil {
			t.Errorf("Failed to clean up test monitor: %v", err)
		}
	})
}

func TestMonitorRepositoryDuplicate(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(db.Close)

	repo := NewMonitorRepository(db)

	testURL := fmt.Sprintf("https://example.com/test-%d", time.Now().UnixNano())

	created, err := repo.Create(ctx, "Test Monitor", testURL, 60)

	if err != nil {
		t.Fatal("Failed to create monitor:", err)
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

	_, err = repo.Create(ctx, "Duplicate Monitor", testURL, 60)

	if err == nil {
		t.Fatal("Expected duplicate URL error, got nil")
	}

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		t.Fatalf("Expected PostgreSQL error, got %v", err)
	}

	if pgErr.Code != "23505" {
		t.Errorf("Expected error code 23505, got %s", pgErr.Code)
	}
}

func TestMonitorRepositoryGetAll(t *testing.T) {

	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(db.Close)

	repo := NewMonitorRepository(db)

	testURL := fmt.Sprintf("https://example.com/test-%d", time.Now().UnixNano())

	created, err := repo.Create(ctx, "Test Monitor", testURL, 60)

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

	monitors, err := repo.GetAll(ctx)

	if err != nil {
		t.Fatal("Failed to retrieve monitors:", err)
	}
	found := false

	for _, m := range monitors {
		if m.ID == created.ID {
			found = true

			if m.Name != created.Name {
				t.Errorf(
					"Expected name %s, got %s",
					created.Name,
					m.Name,
				)
			}

			if m.URL != created.URL {
				t.Errorf(
					"Expected URL %s, got %s",
					created.URL,
					m.URL,
				)
			}
			break
		}
	}

	if !found {
		t.Errorf("Monitor with ID %d was not found", created.ID)
	}
}

func TestMonitorRepositoryGetByID(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := Connect(ctx, databaseURL)
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
		t.Fatalf("Expected pulse_test database, got %s", dbName)
	}

	repo := NewMonitorRepository(db)

	testURL := fmt.Sprintf("https://example.com/test-%d", time.Now().UnixNano())

	created, err := repo.Create(ctx, "Test Monitor", testURL, 60)

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

	retrieved, err := repo.GetByID(ctx, created.ID)

	if err != nil {
		t.Fatal("Failed to retrieve monitor:", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf(
			"Expected ID %d, got %d",
			created.ID,
			retrieved.ID,
		)
	}

	if retrieved.URL != created.URL {
		t.Errorf(
			"Expected URL %s, got %s",
			created.URL,
			retrieved.URL,
		)
	}

	if retrieved.IntervalSeconds != created.IntervalSeconds {
		t.Errorf(
			"Expected interval %d, got %d",
			created.IntervalSeconds,
			retrieved.IntervalSeconds,
		)
	}

	_, err = repo.GetByID(ctx, -1)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("Expected pgx.ErrNoRows, got %v", err)
	}
}
