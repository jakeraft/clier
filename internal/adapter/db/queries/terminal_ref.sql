-- name: SaveTerminalRefs :execresult
INSERT INTO terminal_refs (run_id, team_member_id, refs)
VALUES (?, ?, ?)
ON CONFLICT (run_id, team_member_id) DO UPDATE SET refs = excluded.refs;

-- name: GetTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE run_id = ? AND team_member_id = ?;

-- name: GetRunTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE run_id = ? LIMIT 1;

-- name: DeleteTerminalRefs :execresult
DELETE FROM terminal_refs WHERE run_id = ?;
