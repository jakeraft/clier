-- name: CreateSystemPrompt :execresult
INSERT INTO system_prompts (id, name, prompt, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpsertBuiltInSystemPrompt :execresult
INSERT INTO system_prompts (id, name, prompt, built_in, created_at, updated_at)
VALUES (?, ?, ?, 1, ?, ?)
ON CONFLICT (id) DO UPDATE SET name = excluded.name, prompt = excluded.prompt, updated_at = excluded.updated_at;

-- name: GetSystemPrompt :one
SELECT * FROM system_prompts WHERE id = ?;

-- name: ListSystemPrompts :many
SELECT * FROM system_prompts ORDER BY built_in DESC, created_at;

-- name: ListBuiltInSystemPrompts :many
SELECT * FROM system_prompts WHERE built_in = 1 ORDER BY created_at;

-- name: UpdateSystemPrompt :execresult
UPDATE system_prompts SET name = ?, prompt = ?, updated_at = ? WHERE id = ? AND built_in = 0;

-- name: DeleteSystemPrompt :execresult
DELETE FROM system_prompts WHERE id = ? AND built_in = 0;
