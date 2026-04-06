-- name: CreateSkill :execresult
INSERT INTO skills (id, name, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetSkill :one
SELECT * FROM skills WHERE id = ?;

-- name: ListSkills :many
SELECT * FROM skills ORDER BY created_at;

-- name: UpdateSkill :execresult
UPDATE skills SET name = ?, content = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSkill :execresult
DELETE FROM skills WHERE id = ?;
