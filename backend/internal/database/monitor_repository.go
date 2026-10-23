package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/harimations-git/pulse/backend/internal/monitor"
)

type MonitorRepository struct {
	DB *pgxpool.Pool
}

func NewMonitorRepository(db *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{DB: db}
}

func (repo *MonitorRepository) GetAll(
	ctx context.Context,
) ([]monitor.Monitor, error) {

	// 1. Execute SELECT query
	rows, err := repo.DB.Query(ctx, `
        SELECT id, name, url, interval_seconds, created_at
        FROM monitors
        ORDER BY id
    `)

	// 2. Handle query error
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve monitors: %w", err)
	}

	// 3. Defer closing rows
	defer rows.Close()

	// 4. Create an empty slice

	monitors := []monitor.Monitor{}
	// 5. Loop through rows
	for rows.Next() {

		// Create a Monitor struct
		var m monitor.Monitor
		// Scan row values into the struct
		err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.URL,
			&m.IntervalSeconds,
			&m.CreatedAt,
		)
		// Handle scanning errors
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitor: %w", err)
		}
		// Append monitor to slice
		monitors = append(monitors, m)
	}

	// 6. Check rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate monitors: %w", err)
	}

	// 7. Return the slice and nil error
	return monitors, nil
}
