-- +goose Up
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    duration_ms BIGINT CHECK (duration_ms >= 0),
    reason TEXT NOT NULL CHECK (char_length(reason) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX incidents_one_open_per_monitor_idx
    ON incidents (monitor_id) WHERE resolved_at IS NULL;
CREATE INDEX incidents_monitor_time_idx
    ON incidents (monitor_id, opened_at DESC, id DESC);

-- +goose Down
DROP TABLE incidents;
