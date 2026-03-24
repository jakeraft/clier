-- name: CreateSprintSurface :exec
INSERT INTO sprint_surfaces (sprint_id, member_id, workspace_ref, surface_ref)
VALUES (?, ?, ?, ?);

-- name: GetSprintSurfaceRef :one
SELECT surface_ref FROM sprint_surfaces WHERE sprint_id = ? AND member_id = ?;

-- name: GetSprintWorkspaceRef :one
SELECT workspace_ref FROM sprint_surfaces WHERE sprint_id = ? LIMIT 1;

-- name: DeleteSprintSurfaces :exec
DELETE FROM sprint_surfaces WHERE sprint_id = ?;
