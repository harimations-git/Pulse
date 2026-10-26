package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/harimations-git/pulse/backend/internal/monitor"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCheckRepositorySave(t *testing.T) {
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

	monitorRepo := NewMonitorRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/test-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"Check Repository Test",
		testURL,
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

	// Create a failed monitoring result
	result := monitor.CheckResult{
		URL:          testURL,
		StatusCode:   0,
		ResponseTime: 100 * time.Millisecond,
		Healthy:      false,
		Err:          errors.New("connection timed out"),
	}

	checkRepo := NewCheckRepository(db)

	saved, err := checkRepo.Save(ctx, created.ID, result)

	if err != nil {
		t.Fatal("Failed to save check result:", err)
	}

	// Verify HTTP status
	if saved.StatusCode != 0 {
		t.Errorf("Expected status 0, got %d", saved.StatusCode)
	}

	// Verify unhealthy status
	if saved.Healthy {
		t.Error("Expected service to be unhealthy")
	}

	// Verify response time
	if saved.ResponseTimeMs != 100 {
		t.Errorf(
			"Expected response time 100ms, got %f",
			saved.ResponseTimeMs,
		)
	}

	// Verify error was stored
	if saved.ErrorMessage == nil {
		t.Fatal("Expected an error message, got nil")
	}

	if *saved.ErrorMessage != "connection timed out" {
		t.Errorf(
			"Expected 'connection timed out', got %s",
			*saved.ErrorMessage,
		)
	}

}

func TestCheckRepositorySaveFailure(t *testing.T) {
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

	monitorRepo := NewMonitorRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/test-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"Check Repository Test",
		testURL,
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

	result := monitor.CheckResult{
		URL:          testURL,
		StatusCode:   0,
		ResponseTime: 100 * time.Millisecond,
		Healthy:      false,
		Err:          errors.New("connection timed out"),
	}

	checkRepo := NewCheckRepository(db)

	saved, err := checkRepo.Save(ctx, created.ID, result)

	if err != nil {
		t.Fatal("Failed to save check result:", err)
	}

	if saved.ErrorMessage == nil {
		t.Fatal("Expected an error message, got nil")
	}

	if *saved.ErrorMessage != "connection timed out" {
		t.Errorf(
			"Expected 'connection timed out', got %s",
			*saved.ErrorMessage,
		)
	}
}

func TestCheckRepositoryInvalidMonitor(t *testing.T) {
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

	// Ensure we're using the test database
	var dbName string

	if err := db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName); err != nil {
		t.Fatal("Failed to identify database:", err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	repo := NewCheckRepository(db)

	result := monitor.CheckResult{
		URL:          "https://example.com",
		StatusCode:   200,
		ResponseTime: 150 * time.Millisecond,
		Healthy:      true,
	}

	// -1 is not a valid monitor ID
	_, err = repo.Save(ctx, -1, result)

	// A database error must occur
	if err == nil {
		t.Fatal("Expected foreign key violation, got nil")
	}

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		t.Fatalf("Expected PostgreSQL error, got %v", err)
	}

	if pgErr.Code != "23503" {
		t.Errorf(
			"Expected SQLSTATE 23503, got %s",
			pgErr.Code,
		)
	}
}

func TestCheckRepositoryGetByMonitorID(t *testing.T) {
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

	if err := db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName); err != nil {
		t.Fatal("Failed to identify database:", err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := NewMonitorRepository(db)
	checkRepo := NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/history-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"History Test",
		testURL,
		60,
	)
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

	statuses := []int{200, 503, 201}
	savedIDs := make([]int64, 0, len(statuses))

	for _, status := range statuses {
		result := monitor.CheckResult{
			URL:          testURL,
			StatusCode:   status,
			ResponseTime: 150 * time.Millisecond,
			Healthy:      status >= 200 && status < 300,
		}

		saved, err := checkRepo.Save(ctx, created.ID, result)
		if err != nil {
			t.Fatal("Failed to save check result:", err)
		}

		savedIDs = append(savedIDs, saved.ID)
	}

	records, err := checkRepo.GetByMonitorID(ctx, created.ID)
	if err != nil {
		t.Fatal("Failed to retrieve monitoring history:", err)
	}

	if len(records) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(records))
	}

	for _, record := range records {
		if record.MonitorID != created.ID {
			t.Errorf(
				"Expected monitor ID %d, got %d",
				created.ID,
				record.MonitorID,
			)
		}
	}

	for i := 1; i < len(records); i++ {
		previous := records[i-1]
		current := records[i]

		if previous.CheckedAt.Before(current.CheckedAt) ||
			(previous.CheckedAt.Equal(current.CheckedAt) &&
				previous.ID < current.ID) {
			t.Errorf("Monitoring results are not ordered newest first")
		}
	}

	found := make(map[int64]bool)

	for _, record := range records {
		found[record.ID] = true
	}

	for _, id := range savedIDs {
		if !found[id] {
			t.Errorf("Saved check result %d was not retrieved", id)
		}
	}
}

func TestCheckRepositoryGetLastChecked(t *testing.T) {

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

	if err := db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName); err != nil {
		t.Fatal("Failed to identify database:", err)
	}

	if dbName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", dbName)
	}

	monitorRepo := NewMonitorRepository(db)
	checkRepo := NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/history-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"History Test",
		testURL,
		60,
	)
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

	lastChecked, err := checkRepo.GetLastChecked(ctx, created.ID)

	if err != nil {
		t.Fatal(err)
	}

	if lastChecked != nil {
		t.Errorf("Expected nil, got %v", lastChecked)
	}

	result := monitor.CheckResult{
		URL:          created.URL,
		StatusCode:   200,
		ResponseTime: 150 * time.Millisecond,
		Healthy:      true,
	}

	saved, err := checkRepo.Save(ctx, created.ID, result)
	if err != nil {
		t.Fatal(err)
	}

	lastChecked, err = checkRepo.GetLastChecked(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if lastChecked == nil {
		t.Fatal("Expected a timestamp, got nil")
	}

	if !lastChecked.Equal(saved.CheckedAt) {
		t.Errorf(
			"Expected %v, got %v",
			saved.CheckedAt,
			*lastChecked,
		)
	}
}
