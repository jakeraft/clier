-- name: SaveSessionSurface :execresult
INSERT INTO session_surfaces (session_id, member_id, workspace_ref, surface_ref)
VALUES (?, ?, ?, ?);

-- name: GetSessionSurface :one
SELECT workspace_ref, surface_ref FROM session_surfaces
WHERE session_id = ? AND member_id = ?;

-- name: GetSessionWorkspaceRef :one
SELECT workspace_ref FROM session_surfaces
WHERE session_id = ? AND member_id != ? LIMIT 1;

-- name: DeleteSessionSurfaces :execresult
DELETE FROM session_surfaces WHERE session_id = ?;
