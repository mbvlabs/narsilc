-- name: ListAuthors :many
-- @filter missing eq
SELECT id, name FROM authors;
