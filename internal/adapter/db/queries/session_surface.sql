-- name: SaveSessionSurface :execresult
INSERT INTO session_surfaces (session_id, team_member_id, workspace_ref, surface_ref)
VALUES (?, ?, ?, ?);

-- name: GetSessionSurface :one
SELECT workspace_ref, surface_ref FROM session_surfaces
WHERE session_id = ? AND team_member_id = ?;

-- name: GetSessionWorkspaceRef :one
SELECT workspace_ref FROM session_surfaces
WHERE session_id = ? LIMIT 1;

-- name: DeleteSessionSurfaces :execresult
DELETE FROM session_surfaces WHERE session_id = ?;
