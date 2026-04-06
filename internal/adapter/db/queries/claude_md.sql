-- name: CreateClaudeMd :execresult
INSERT INTO claude_mds (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetClaudeMd :one
SELECT * FROM claude_mds WHERE id = ?;

-- name: ListClaudeMds :many
SELECT * FROM claude_mds ORDER BY created_at;

-- name: UpdateClaudeMd :execresult
UPDATE claude_mds SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteClaudeMd :execresult
DELETE FROM claude_mds WHERE id = ?;
