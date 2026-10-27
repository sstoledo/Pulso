-- name: CreateMembership :one
INSERT INTO memberships (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListOrganizationsByUser :many
SELECT o.id, o.name, o.created_at, m.role
FROM memberships AS m
JOIN organizations AS o ON o.id = m.organization_id
WHERE m.user_id = $1
ORDER BY o.created_at, o.id;

-- name: GetOrganizationMembership :one
SELECT o.id, o.name, o.created_at, m.role
FROM memberships AS m
JOIN organizations AS o ON o.id = m.organization_id
WHERE m.user_id = $1
  AND m.organization_id = $2;
