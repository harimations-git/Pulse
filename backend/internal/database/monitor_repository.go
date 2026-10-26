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

	rows, err := repo.DB.Query(ctx, `
        SELECT id, name, url, interval_seconds, created_at
        FROM monitors
        ORDER BY id
    `)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve monitors: %w", err)
	}

	defer rows.Close()

	monitors := []monitor.Monitor{}

	for rows.Next() {

		var m monitor.Monitor

		err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.URL,
			&m.IntervalSeconds,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitor: %w", err)
		}
		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate monitors: %w", err)
	}

	return monitors, nil
}

func (repo *MonitorRepository) Create(
	ctx context.Context,
	name string,
	url string,
	intervalSeconds int,
) (monitor.Monitor, error) {

	var m monitor.Monitor

	row := repo.DB.QueryRow(ctx, `
		INSERT INTO monitors (name, url, interval_seconds)
		VALUES ($1, $2, $3)
		RETURNING id, name, url, interval_seconds, created_at
	`, name, url, intervalSeconds)

	err := row.Scan(
		&m.ID,
		&m.Name,
		&m.URL,
		&m.IntervalSeconds,
		&m.CreatedAt,
	)

	if err != nil {
		return monitor.Monitor{},
			fmt.Errorf("failed to create monitor: %w", err)
	}

	return m, nil
}

func (repo *MonitorRepository) GetByID(ctx context.Context, id int64) (monitor.Monitor, error) {

	var m monitor.Monitor

	err := repo.DB.QueryRow(ctx, `
	        SELECT
            id,
            name,
            url,
            interval_seconds,
            created_at
			FROM monitors
			WHERE id = $1
	`, id).Scan(
		&m.ID,
		&m.Name,
		&m.URL,
		&m.IntervalSeconds,
		&m.CreatedAt,
	)

	if err != nil {
		return monitor.Monitor{}, fmt.Errorf(
			"failed to retrieve monitor: %w", err,
		)
	}

	return m, nil
}
