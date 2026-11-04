package database

import (
	"context"
	"fmt"
	"time"

	"github.com/harimations-git/pulse/backend/internal/monitor"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckRepository struct {
	DB *pgxpool.Pool
}

func NewCheckRepository(db *pgxpool.Pool) *CheckRepository {
	return &CheckRepository{DB: db}
}

func (repo *CheckRepository) Save(
	ctx context.Context,
	monitorID int64,
	result monitor.CheckResult,
) (monitor.CheckRecord, error) {

	// Convert response time to milliseconds
	responseTimeMs := result.ResponseTime.Seconds() * 1000

	// Convert Go error into nullable database value
	var errorMessage *string

	if result.Err != nil {
		message := result.Err.Error()
		errorMessage = &message
	}

	// Prepare the record
	var record monitor.CheckRecord

	// Insert the monitoring result
	row := repo.DB.QueryRow(ctx, `
		INSERT INTO check_results (
			monitor_id,
			status_code,
			response_time_ms,
			healthy,
			error_message
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			monitor_id,
			status_code,
			response_time_ms,
			healthy,
			error_message,
			checked_at
	`,
		monitorID,
		result.StatusCode,
		responseTimeMs,
		result.Healthy,
		errorMessage,
	)

	// Map returned database values into the struct
	err := row.Scan(
		&record.ID,
		&record.MonitorID,
		&record.StatusCode,
		&record.ResponseTimeMs,
		&record.Healthy,
		&record.ErrorMessage,
		&record.CheckedAt,
	)

	// Handle database errors
	if err != nil {
		return monitor.CheckRecord{},
			fmt.Errorf("failed to save check result: %w", err)
	}

	return record, nil
}

func (repo *CheckRepository) GetByMonitorID(ctx context.Context, monitorID int64) ([]monitor.CheckRecord, error) {
	rows, err := repo.DB.Query(ctx, `
	        SELECT
            id,
            monitor_id,
            status_code,
            response_time_ms,
            healthy,
            error_message,
            checked_at
        FROM check_results
        WHERE monitor_id = $1
        ORDER BY checked_at DESC, id DESC
		LIMIT 30
	`, monitorID)

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve check results: %w", err)
	}

	defer rows.Close()

	records := []monitor.CheckRecord{}

	for rows.Next() {
		var record monitor.CheckRecord

		err := rows.Scan(
			&record.ID,
			&record.MonitorID,
			&record.StatusCode,
			&record.ResponseTimeMs,
			&record.Healthy,
			&record.ErrorMessage,
			&record.CheckedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("Failed to scan check result: %w", err)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to iterate check results: %w", err)
	}

	return records, nil
}

func (repo *CheckRepository) GetLastChecked(
	ctx context.Context, monitorID int64,
) (*time.Time, error) {
	var lastChecked *time.Time

	err := repo.DB.QueryRow(ctx, `
	    SELECT MAX(checked_at)
        FROM check_results
        WHERE monitor_id = $1
	`, monitorID).Scan(&lastChecked)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to retrieve last check time: %w",
			err,
		)
	}

	return lastChecked, nil
}

func (r *CheckRepository) GetStatsByMonitorID(
	ctx context.Context,
	monitorID int64,
) (monitor.Stats, error) {
	stats := monitor.Stats{
		MonitorID: monitorID,
	}

	query := `
        SELECT
            COUNT(*),
            COUNT(*) FILTER (WHERE healthy),
            COALESCE(AVG(response_time_ms), 0)::double precision
        FROM check_results
        WHERE monitor_id = $1
    `

	err := r.DB.QueryRow(
		ctx,
		query,
		monitorID,
	).Scan(
		&stats.TotalChecks,
		&stats.SuccessfulChecks,
		&stats.AverageResponseTimeMs,
	)

	if err != nil {
		return monitor.Stats{}, err
	}

	// Avoid dividing by zero when a monitor
	// has no recorded checks.
	if stats.TotalChecks > 0 {
		stats.UptimePercentage =
			float64(stats.SuccessfulChecks) /
				float64(stats.TotalChecks) * 100
	}

	return stats, nil
}
