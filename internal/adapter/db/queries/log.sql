-- name: CreateLog :execresult
INSERT INTO logs (id, session_id, team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListLogsBySessionID :many
SELECT * FROM logs WHERE session_id = ? ORDER BY created_at;
