-- +goose Up
CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    url TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds IN (30, 60, 300)),
    timeout_ms INTEGER NOT NULL CHECK (timeout_ms BETWEEN 100 AND 10000),
    expected_status_min INTEGER NOT NULL DEFAULT 200 CHECK (expected_status_min BETWEEN 100 AND 599),
    expected_status_max INTEGER NOT NULL DEFAULT 399 CHECK (expected_status_max BETWEEN 100 AND 599),
    enabled BOOLEAN NOT NULL DEFAULT true,
    show_public BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'UNKNOWN' CHECK (status IN ('UNKNOWN', 'UP', 'DOWN')),
    consecutive_failures INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    last_check_at TIMESTAMPTZ,
    last_latency_ms INTEGER CHECK (last_latency_ms >= 0),
    next_check_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expected_status_min <= expected_status_max)
);

CREATE INDEX monitors_organization_page_idx
    ON monitors (organization_id, created_at DESC, id DESC);
CREATE INDEX monitors_next_check_idx
    ON monitors (next_check_at) WHERE enabled;

-- +goose Down
DROP TABLE monitors;
