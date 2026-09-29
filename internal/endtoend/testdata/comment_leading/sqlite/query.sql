-- File-level notes, not a query doc comment.

-- name: ListAuthors :many
-- Returns all authors.
SELECT id, name FROM authors;

-- name: CountAuthors :one
SELECT count(*) FROM authors;
