-- name: SaveTerminalRefs :execresult
INSERT INTO terminal_refs (task_id, team_member_id, refs)
VALUES (?, ?, ?)
ON CONFLICT (task_id, team_member_id) DO UPDATE SET refs = excluded.refs;

-- name: GetTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE task_id = ? AND team_member_id = ?;

-- name: GetTaskTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE task_id = ? LIMIT 1;

-- name: DeleteTerminalRefs :execresult
DELETE FROM terminal_refs WHERE task_id = ?;
