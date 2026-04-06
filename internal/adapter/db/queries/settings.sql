-- name: CreateSettings :execresult
INSERT INTO settings (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetSettings :one
SELECT * FROM settings WHERE id = ?;

-- name: ListSettings :many
SELECT * FROM settings ORDER BY created_at;

-- name: UpdateSettings :execresult
UPDATE settings SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSettings :execresult
DELETE FROM settings WHERE id = ?;
