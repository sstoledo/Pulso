-- +goose Up
CREATE TABLE notification_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('OPENED', 'RECOVERED')),
    idempotency_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 10),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    last_error TEXT CHECK (char_length(last_error) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);

CREATE INDEX notification_jobs_claim_idx
    ON notification_jobs (status, next_attempt_at);
CREATE INDEX notification_jobs_stale_idx
    ON notification_jobs (status, locked_at) WHERE status = 'pending';

CREATE TABLE notification_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES notification_jobs(id) ON DELETE CASCADE,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    error TEXT CHECK (char_length(error) <= 500)
);

CREATE INDEX notification_attempts_job_time_idx
    ON notification_attempts (job_id, attempted_at DESC, id DESC);

-- +goose Down
DROP TABLE notification_attempts;
DROP TABLE notification_jobs;
