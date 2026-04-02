-- name: CreateSprint :execresult
INSERT INTO sprints (id, name, snapshot, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetSprint :one
SELECT * FROM sprints WHERE id = ?;

-- name: ListSprints :many
SELECT * FROM sprints ORDER BY created_at;

-- name: DeleteSprint :execresult
DELETE FROM sprints WHERE id = ?;
