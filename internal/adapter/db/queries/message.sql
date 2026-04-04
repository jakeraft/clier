-- name: CreateMessage :execresult
INSERT INTO messages (id, session_id, from_team_member_id, to_team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMessagesBySessionID :many
SELECT * FROM messages WHERE session_id = ? ORDER BY created_at;

-- name: ListMessagesBySessionAndMember :many
SELECT * FROM messages
WHERE session_id = ? AND (from_team_member_id = ? OR to_team_member_id = ?)
ORDER BY created_at;
