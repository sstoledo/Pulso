-- +goose Up
CREATE TABLE status_pages (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 100),
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (char_length(slug) BETWEEN 3 AND 64),
    CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])?$')
);

CREATE INDEX status_pages_published_slug_idx ON status_pages (slug) WHERE is_published;

-- +goose Down
DROP TABLE status_pages;
