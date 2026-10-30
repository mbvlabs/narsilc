-- name: ListPairs :many
SELECT a.id, b.id
FROM left_items a
JOIN right_items b ON b.left_id = a.id
ORDER BY a.id;
