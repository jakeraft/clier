-- name: CreateClaudeJson :execresult
INSERT INTO claude_jsons (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetClaudeJson :one
SELECT * FROM claude_jsons WHERE id = ?;

-- name: ListClaudeJsons :many
SELECT * FROM claude_jsons ORDER BY created_at;

-- name: UpdateClaudeJson :execresult
UPDATE claude_jsons SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteClaudeJson :execresult
DELETE FROM claude_jsons WHERE id = ?;
