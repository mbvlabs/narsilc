-- name: CountArg :one
SELECT count(nope1) FROM users;

-- name: LowerArg :many
SELECT lower(nope2) FROM users;

-- name: Nested :many
SELECT upper(lower(nope3)) FROM users;

-- name: CaseWhen :many
SELECT CASE WHEN nope4 > 1 THEN 'a' ELSE 'b' END AS x FROM users;

-- name: CoalesceNested :many
SELECT coalesce(lower(nope5), 'x') AS x FROM users;

-- name: CastArg :many
SELECT CAST(nope6 AS SIGNED) AS x FROM users;

-- name: WhereNoParam :many
SELECT id FROM users WHERE nope7 = 1;

-- name: OpTarget :many
SELECT id + nope8 AS x FROM users;

-- name: OrderByFunc :many
SELECT id FROM users ORDER BY lower(nope9);

-- name: WhereFunc :many
SELECT id FROM users WHERE lower(nope10) = sqlc.arg(email);

-- name: Having :many
SELECT name FROM users GROUP BY name HAVING count(nope11) > 1;

-- name: UpdateSet :exec
UPDATE users SET name = lower(nope12) WHERE id = sqlc.arg(id);

-- name: Subquery :many
SELECT id FROM users WHERE id IN (SELECT user_id FROM posts WHERE lower(nope13) = 'x');

-- name: JoinOn :many
SELECT users.id FROM users JOIN posts ON posts.nope14 = users.id;

-- name: BadAlias :many
SELECT lower(nosuch.email) FROM users;

-- name: Valid :many
SELECT count(id) AS cnt, lower(email) AS le, coalesce(name, 'x') AS n,
  CASE WHEN id > 1 THEN 'a' ELSE 'b' END AS c, CAST(id AS SIGNED) AS t,
  (SELECT count(*) FROM posts p WHERE p.user_id = users.id) AS post_count
FROM users
WHERE lower(email) = sqlc.arg(email)
  AND EXISTS (SELECT 1 FROM posts WHERE posts.user_id = users.id AND lower(title) <> '')
GROUP BY id, email, name
ORDER BY lower(email), cnt;

-- name: ValidCTE :many
WITH x AS (SELECT user_id, count(*) AS n FROM posts GROUP BY user_id)
SELECT u.id, coalesce(x.n, 0) AS n FROM users u LEFT JOIN x ON x.user_id = u.id WHERE upper(u.email) <> '';

-- name: ValidDerived :many
SELECT s.total FROM (SELECT count(*) AS total FROM posts) s WHERE abs(s.total) > 0;
