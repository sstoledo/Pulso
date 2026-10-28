-- +goose Up
CREATE TABLE check_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_job_id UUID NOT NULL,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    checked_at TIMESTAMPTZ NOT NULL,
    success BOOLEAN NOT NULL,
    http_status INTEGER CHECK (http_status BETWEEN 100 AND 599),
    latency_ms INTEGER NOT NULL CHECK (latency_ms >= 0),
    error_summary TEXT CHECK (char_length(error_summary) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (check_job_id),
    FOREIGN KEY (check_job_id, monitor_id) REFERENCES check_jobs(id, monitor_id) ON DELETE CASCADE
);

CREATE INDEX check_results_monitor_time_idx
    ON check_results (monitor_id, checked_at DESC, id DESC);

-- +goose Down
DROP TABLE check_results;
