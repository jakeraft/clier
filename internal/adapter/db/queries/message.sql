-- name: CreateMessage :execresult
INSERT INTO messages (id, task_id, from_team_member_id, to_team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMessagesByTaskID :many
SELECT * FROM messages WHERE task_id = ? ORDER BY created_at;

-- name: ListMessagesByTaskAndMember :many
SELECT * FROM messages
WHERE task_id = ? AND (from_team_member_id = ? OR to_team_member_id = ?)
ORDER BY created_at;
