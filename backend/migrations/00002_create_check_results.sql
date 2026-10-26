
-- +goose Up

CREATE TABLE check_results (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    monitor_id BIGINT NOT NULL
        REFERENCES monitors(id) ON DELETE CASCADE,

    status_code INTEGER NOT NULL,
    response_time_ms DOUBLE PRECISION NOT NULL,
    healthy BOOLEAN NOT NULL,

    error_message TEXT,

    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_response_time
        CHECK (response_time_ms >= 0)
);

CREATE INDEX idx_check_results_monitor_time
ON check_results (monitor_id, checked_at DESC);

-- +goose Down

DROP TABLE IF EXISTS check_results;
