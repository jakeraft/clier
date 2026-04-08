-- name: CreateRun :execresult
INSERT INTO runs (id, name, team_id, status, plan, started_at, stopped_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetRun :one
SELECT * FROM runs WHERE id = ?;

-- name: ListRuns :many
SELECT * FROM runs ORDER BY started_at DESC;

-- name: UpdateRunStatus :execresult
UPDATE runs SET status = ?, stopped_at = ? WHERE id = ?;

-- name: DeleteRun :execresult
DELETE FROM runs WHERE id = ?;
