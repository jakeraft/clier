-- name: CreateSession :execresult
INSERT INTO sessions (id, team_id, status, created_at, stopped_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions ORDER BY created_at;

-- name: UpdateSessionStatus :execresult
UPDATE sessions SET status = ?, stopped_at = ? WHERE id = ?;

-- name: DeleteSession :execresult
DELETE FROM sessions WHERE id = ?;
