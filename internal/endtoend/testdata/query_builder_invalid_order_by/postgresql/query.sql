-- name: ListAuthors :many
-- @order name
SELECT id, name FROM authors
ORDER BY name;
