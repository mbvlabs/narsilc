-- name: ListUsers :many
-- @order public.users.id
SELECT u.id, u.email FROM users u;
