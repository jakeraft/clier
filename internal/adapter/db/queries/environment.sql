-- name: CreateEnvironment :execresult
INSERT INTO environments (id, name, key, value, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetEnvironment :one
SELECT * FROM environments WHERE id = ?;

-- name: ListEnvironments :many
SELECT * FROM environments ORDER BY created_at;

-- name: UpdateEnvironment :execresult
UPDATE environments SET name = ?, key = ?, value = ?, updated_at = ? WHERE id = ?;

-- name: DeleteEnvironment :execresult
DELETE FROM environments WHERE id = ?;
