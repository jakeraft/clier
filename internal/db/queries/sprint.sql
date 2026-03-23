-- name: CreateSprint :exec
INSERT INTO sprints (id, name, team_snapshot, state, error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetSprint :one
SELECT * FROM sprints WHERE id = ?;

-- name: ListSprints :many
SELECT * FROM sprints ORDER BY created_at;

-- name: UpdateSprintState :exec
UPDATE sprints SET state = ?, error = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSprint :exec
DELETE FROM sprints WHERE id = ?;
