-- +goose Up
CREATE TABLE check_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    scheduled_for TIMESTAMPTZ NOT NULL,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed')),
    locked_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (monitor_id, scheduled_for),
    UNIQUE (id, monitor_id)
);

CREATE UNIQUE INDEX check_jobs_one_active_per_monitor_idx
    ON check_jobs (monitor_id) WHERE status IN ('pending', 'running');
CREATE INDEX check_jobs_claim_idx
    ON check_jobs (status, available_at, scheduled_for);
CREATE INDEX check_jobs_stale_idx
    ON check_jobs (status, locked_at) WHERE status = 'running';

-- +goose Down
DROP TABLE check_jobs;
