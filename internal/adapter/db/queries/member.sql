-- name: CreateMember :execresult
INSERT INTO members (id, name, model, args, claude_md_id, settings_id, claude_json_id, git_repo_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetMember :one
SELECT * FROM members WHERE id = ?;

-- name: ListMembers :many
SELECT * FROM members ORDER BY created_at;

-- name: UpdateMember :execresult
UPDATE members SET name = ?, model = ?, args = ?, claude_md_id = ?, settings_id = ?, claude_json_id = ?, git_repo_id = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMember :execresult
DELETE FROM members WHERE id = ?;

-- name: AddMemberSkill :execresult
INSERT INTO member_skills (member_id, skill_id) VALUES (?, ?);

-- name: RemoveMemberSkill :execresult
DELETE FROM member_skills WHERE member_id = ? AND skill_id = ?;

-- name: ListMemberSkillIDs :many
SELECT skill_id FROM member_skills WHERE member_id = ? ORDER BY rowid;

-- name: DeleteMemberSkills :execresult
DELETE FROM member_skills WHERE member_id = ?;

