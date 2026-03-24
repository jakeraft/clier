-- name: CreateSystemPrompt :execresult
INSERT INTO system_prompts (id, name, prompt, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetSystemPrompt :one
SELECT * FROM system_prompts WHERE id = ?;

-- name: ListSystemPrompts :many
SELECT * FROM system_prompts ORDER BY created_at;

-- name: UpdateSystemPrompt :execresult
UPDATE system_prompts SET name = ?, prompt = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSystemPrompt :execresult
DELETE FROM system_prompts WHERE id = ?;
