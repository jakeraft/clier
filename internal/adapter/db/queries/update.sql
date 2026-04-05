-- name: CreateUpdate :execresult
INSERT INTO updates (id, task_id, team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListUpdatesByTaskID :many
SELECT * FROM updates WHERE task_id = ? ORDER BY created_at;
