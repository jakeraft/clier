-- name: CreateMessage :execresult
INSERT INTO messages (id, run_id, from_team_member_id, to_team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMessagesByRunID :many
SELECT * FROM messages WHERE run_id = ? ORDER BY created_at;

-- name: ListMessagesByRunAndMember :many
SELECT * FROM messages
WHERE run_id = ? AND (from_team_member_id = ? OR to_team_member_id = ?)
ORDER BY created_at;
