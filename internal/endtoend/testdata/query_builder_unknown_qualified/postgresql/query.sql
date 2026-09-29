-- name: ListUserTokens :many
-- @order u.missing
SELECT u.id, u.email, t.scope
FROM users u
JOIN tokens t ON t.user_id = u.id;
