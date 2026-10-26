package scheduler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/harimations-git/pulse/backend/internal/database"
	"github.com/harimations-git/pulse/backend/internal/monitor"
)

func NewScheduledCheck(
	checkRepo *database.CheckRepository,
	client *http.Client,
) CheckFunc {

	return func(ctx context.Context, m monitor.Monitor) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		result := monitor.CheckService(m.URL, client)

		saved, err := checkRepo.Save(ctx, m.ID, result)
		if err != nil {
			return fmt.Errorf(
				"failed to save check for monitor %d: %w",
				m.ID,
				err,
			)
		}

		fmt.Printf(
			"Checked %s: status=%d healthy=%t response=%.2fms\n",
			m.Name,
			saved.StatusCode,
			saved.Healthy,
			saved.ResponseTimeMs,
		)

		return nil
	}
}
