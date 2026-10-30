-- name: ListPairs :many
SELECT a.id AS left_id, b.id AS right_id
FROM left_items a
JOIN right_items b ON b.left_id = a.id
ORDER BY a.id;
