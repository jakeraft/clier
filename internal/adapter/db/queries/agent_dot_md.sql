-- name: CreateAgentDotMd :execresult
INSERT INTO agent_dot_mds (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetAgentDotMd :one
SELECT * FROM agent_dot_mds WHERE id = ?;

-- name: ListAgentDotMds :many
SELECT * FROM agent_dot_mds ORDER BY created_at;

-- name: UpdateAgentDotMd :execresult
UPDATE agent_dot_mds SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteAgentDotMd :execresult
DELETE FROM agent_dot_mds WHERE id = ?;
