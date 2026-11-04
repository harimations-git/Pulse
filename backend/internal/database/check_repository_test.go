package database

import (
	"context"
	"errors"
	"fmt"
	"math"
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

func TestGetStatsByMonitorID(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal("Failed to connect to test database:", err)
	}
	t.Cleanup(db.Close)

	var databaseName string
	err = db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&databaseName)

	if err != nil {
		t.Fatal(err)
	}

	if databaseName != "pulse_test" {
		t.Fatalf("Expected pulse_test, got %s", databaseName)
	}

	monitorRepo := NewMonitorRepository(db)
	checkRepo := NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://pulse-stats-%d.example.com",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"Statistics Test",
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

	t.Run("No checks", func(t *testing.T) {
		stats, err := checkRepo.GetStatsByMonitorID(
			ctx,
			created.ID,
		)
		if err != nil {
			t.Fatal(err)
		}

		if stats.TotalChecks != 0 ||
			stats.SuccessfulChecks != 0 ||
			stats.UptimePercentage != 0 ||
			stats.AverageResponseTimeMs != 0 {
			t.Errorf("Expected zero statistics, got %+v", stats)
		}
	})

	checks := []monitor.CheckResult{
		{URL: testURL, StatusCode: 200, ResponseTime: 100 * time.Millisecond, Healthy: true},
		{URL: testURL, StatusCode: 200, ResponseTime: 150 * time.Millisecond, Healthy: true},
		{URL: testURL, StatusCode: 503, ResponseTime: 300 * time.Millisecond, Healthy: false},
		{URL: testURL, StatusCode: 200, ResponseTime: 200 * time.Millisecond, Healthy: true},
		{URL: testURL, StatusCode: 200, ResponseTime: 250 * time.Millisecond, Healthy: true},
	}

	for _, check := range checks {
		if _, err := checkRepo.Save(
			ctx,
			created.ID,
			check,
		); err != nil {
			t.Fatal("Failed to save check:", err)
		}
	}
	t.Run("Five checks", func(t *testing.T) {
		stats, err := checkRepo.GetStatsByMonitorID(
			ctx,
			created.ID,
		)
		if err != nil {
			t.Fatal(err)
		}

		if stats.MonitorID != created.ID {
			t.Errorf("Incorrect monitor ID: %d", stats.MonitorID)
		}

		if stats.TotalChecks != 5 {
			t.Errorf("Expected 5 checks, got %d", stats.TotalChecks)
		}

		if stats.SuccessfulChecks != 4 {
			t.Errorf("Expected 4 successful checks, got %d",
				stats.SuccessfulChecks)
		}

		if stats.UptimePercentage != 80 {
			t.Errorf("Expected 80%% uptime, got %.2f%%",
				stats.UptimePercentage)
		}

		if math.Abs(stats.AverageResponseTimeMs-200) > 0.001 {
			t.Errorf("Expected 200ms average, got %.2fms",
				stats.AverageResponseTimeMs)
		}
	})
}

func TestCheckRepositoryHistoryLimit(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	db, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

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

	monitorRepo := NewMonitorRepository(db)
	checkRepo := NewCheckRepository(db)

	testURL := fmt.Sprintf(
		"https://example.com/history-limit-%d",
		time.Now().UnixNano(),
	)

	created, err := monitorRepo.Create(
		ctx,
		"History Limit Test",
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

	savedIDs := make([]int64, 0, 35)

	for i := 0; i < 35; i++ {
		result := monitor.CheckResult{
			URL:          testURL,
			StatusCode:   200,
			ResponseTime: time.Duration(100+i) * time.Millisecond,
			Healthy:      true,
		}

		saved, err := checkRepo.Save(
			ctx,
			created.ID,
			result,
		)
		if err != nil {
			t.Fatal(err)
		}

		savedIDs = append(savedIDs, saved.ID)
	}

	records, err := checkRepo.GetByMonitorID(
		ctx,
		created.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 30 {
		t.Fatalf(
			"Expected 30 results, got %d",
			len(records),
		)
	}

	for i, record := range records {
		expectedID := savedIDs[34-i]

		if record.ID != expectedID {
			t.Errorf(
				"At position %d: expected ID %d, got %d",
				i,
				expectedID,
				record.ID,
			)
		}
	}
}
