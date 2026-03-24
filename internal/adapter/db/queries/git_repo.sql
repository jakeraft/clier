-- name: CreateGitRepo :execresult
INSERT INTO git_repos (id, name, url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetGitRepo :one
SELECT * FROM git_repos WHERE id = ?;

-- name: ListGitRepos :many
SELECT * FROM git_repos ORDER BY created_at;

-- name: UpdateGitRepo :execresult
UPDATE git_repos SET name = ?, url = ?, updated_at = ? WHERE id = ?;

-- name: DeleteGitRepo :execresult
DELETE FROM git_repos WHERE id = ?;
