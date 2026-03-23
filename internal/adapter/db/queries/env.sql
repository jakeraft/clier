-- name: CreateEnv :execresult
INSERT INTO envs (id, name, key, value, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetEnv :one
SELECT * FROM envs WHERE id = ?;

-- name: ListEnvs :many
SELECT * FROM envs ORDER BY created_at;

-- name: UpdateEnv :execresult
UPDATE envs SET name = ?, key = ?, value = ?, updated_at = ? WHERE id = ?;

-- name: DeleteEnv :execresult
DELETE FROM envs WHERE id = ?;
