-- name: ListUserTokens :many
-- @filter u.id eq
-- @filter t.id eq
-- @filter scope eq
-- @order  u.id, t.id, scope
SELECT u.id, u.email, t.id, t.scope
FROM users u
JOIN tokens t ON t.user_id = u.id;

-- name: ListActiveUsers :many
-- @order u.id
WITH u AS (
  SELECT id, email FROM users
)
SELECT id, email FROM u;

-- name: ListUsers :many
-- @order public.users.id
SELECT id, email FROM users;
