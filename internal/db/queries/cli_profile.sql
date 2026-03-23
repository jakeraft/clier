-- name: CreateCliProfile :exec
INSERT INTO cli_profiles (id, name, model, binary, system_args, custom_args, dot_config, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCliProfile :one
SELECT * FROM cli_profiles WHERE id = ?;

-- name: ListCliProfiles :many
SELECT * FROM cli_profiles ORDER BY created_at;

-- name: UpdateCliProfile :exec
UPDATE cli_profiles
SET name = ?, model = ?, binary = ?, system_args = ?, custom_args = ?, dot_config = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteCliProfile :exec
DELETE FROM cli_profiles WHERE id = ?;
