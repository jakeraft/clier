-- name: CreateTask :execresult
INSERT INTO tasks (id, name, team_id, status, plan, created_at, stopped_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY created_at DESC;

-- name: UpdateTaskStatus :execresult
UPDATE tasks SET status = ?, stopped_at = ? WHERE id = ?;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = ?;
