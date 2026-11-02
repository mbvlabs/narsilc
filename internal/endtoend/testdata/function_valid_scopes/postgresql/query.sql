-- name: ValidScopes :many
SELECT lower(u.name) AS normalized,
       (SELECT count(lower(u.name)) FROM users AS inner_user WHERE inner_user.parent_id = u.id) AS matches
FROM users AS u LEFT JOIN users AS parent ON parent.id = u.parent_id
ORDER BY normalized;
