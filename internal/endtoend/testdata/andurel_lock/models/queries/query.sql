-- name: GetAuthor :one
SELECT * FROM authors
WHERE id = $1 LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (
  name, bio
) VALUES (
  $1, $2
)
RETURNING *;

-- name: DeleteAuthor :exec
DELETE FROM authors
WHERE id = $1;

-- name: CountAuthors :one
SELECT count(*) FROM authors;

-- name: AuthorPostCounts :many
SELECT a.id, a.name, count(p.id) AS post_count
FROM authors a
LEFT JOIN posts p ON p.author_id = a.id
GROUP BY a.id, a.name;
