-- name: CreateClaudeSettings :execresult
INSERT INTO claude_settings (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetClaudeSettings :one
SELECT * FROM claude_settings WHERE id = ?;

-- name: ListClaudeSettings :many
SELECT * FROM claude_settings ORDER BY created_at;

-- name: UpdateClaudeSettings :execresult
UPDATE claude_settings SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteClaudeSettings :execresult
DELETE FROM claude_settings WHERE id = ?;
