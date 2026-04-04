-- name: SaveTerminalRefs :execresult
INSERT INTO terminal_refs (session_id, team_member_id, refs)
VALUES (?, ?, ?)
ON CONFLICT (session_id, team_member_id) DO UPDATE SET refs = excluded.refs;

-- name: GetTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE session_id = ? AND team_member_id = ?;

-- name: GetSessionTerminalRefs :one
SELECT refs FROM terminal_refs
WHERE session_id = ? LIMIT 1;

-- name: DeleteTerminalRefs :execresult
DELETE FROM terminal_refs WHERE session_id = ?;
