-- name: CreateMember :execresult
INSERT INTO members (id, name, cli_profile_id, git_repo_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetMember :one
SELECT * FROM members WHERE id = ?;

-- name: ListMembers :many
SELECT * FROM members ORDER BY created_at;

-- name: UpdateMember :execresult
UPDATE members SET name = ?, cli_profile_id = ?, git_repo_id = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMember :execresult
DELETE FROM members WHERE id = ?;

-- name: AddMemberSystemPrompt :execresult
INSERT INTO member_system_prompts (member_id, system_prompt_id) VALUES (?, ?);

-- name: RemoveMemberSystemPrompt :execresult
DELETE FROM member_system_prompts WHERE member_id = ? AND system_prompt_id = ?;

-- name: ListMemberSystemPromptIDs :many
SELECT system_prompt_id FROM member_system_prompts WHERE member_id = ?;

-- name: DeleteMemberSystemPrompts :execresult
DELETE FROM member_system_prompts WHERE member_id = ?;

-- name: AddMemberEnvironment :execresult
INSERT INTO member_environments (member_id, environment_id) VALUES (?, ?);

-- name: RemoveMemberEnvironment :execresult
DELETE FROM member_environments WHERE member_id = ? AND environment_id = ?;

-- name: ListMemberEnvironmentIDs :many
SELECT environment_id FROM member_environments WHERE member_id = ?;

-- name: DeleteMemberEnvironments :execresult
DELETE FROM member_environments WHERE member_id = ?;
