-- name: BadCount :one
SELECT count(nonexistent_col) FROM users;

-- name: BadLower :one
SELECT lower(u.nonexistent_col) FROM users AS u;

-- name: BadAlias :one
SELECT lower(nonexistent_col) AS nonexistent_col FROM users;
