-- name: CreateCliProfile :execresult
INSERT INTO cli_profiles (id, name, model, binary, system_args, custom_args, settings_json, claude_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCliProfile :one
SELECT * FROM cli_profiles WHERE id = ?;

-- name: ListCliProfiles :many
SELECT * FROM cli_profiles ORDER BY created_at;

-- name: UpdateCliProfile :execresult
UPDATE cli_profiles
SET name = ?, model = ?, binary = ?, system_args = ?, custom_args = ?, settings_json = ?, claude_json = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteCliProfile :execresult
DELETE FROM cli_profiles WHERE id = ?;
