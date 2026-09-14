-- name: ListAuthors :many
-- @filter name eq,like
-- @filter created_at gte,lte
-- @order  name, created_at, id
SELECT id, name, created_at FROM authors
WHERE tenant_id = sqlc.arg('tenant_id');
