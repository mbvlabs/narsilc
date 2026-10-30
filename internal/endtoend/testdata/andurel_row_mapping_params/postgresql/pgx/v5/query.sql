-- name: CreateEvent :one
INSERT INTO events (
  release_id, occurred_at, archived_at, note
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: TouchEvent :exec
UPDATE events
SET occurred_at = $1
WHERE id = $2;

-- name: ListByOccurredAt :many
SELECT * FROM events
WHERE occurred_at = $1
ORDER BY id;

-- name: ListByRelease :many
SELECT * FROM events
WHERE release_id = $1
ORDER BY id;

-- name: ListByReleaseCast :many
SELECT * FROM events
WHERE release_id = @release_id::bigint
ORDER BY id;
