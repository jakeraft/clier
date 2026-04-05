-- name: CreateNote :execresult
INSERT INTO notes (id, task_id, team_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListNotesByTaskID :many
SELECT * FROM notes WHERE task_id = ? ORDER BY created_at;
