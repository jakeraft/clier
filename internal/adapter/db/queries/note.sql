-- name: CreateNote :execresult
INSERT INTO notes (id, run_id, team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListNotesByRunID :many
SELECT * FROM notes WHERE run_id = ? ORDER BY created_at;
