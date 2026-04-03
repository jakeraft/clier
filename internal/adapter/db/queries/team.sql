-- name: CreateTeam :execresult
INSERT INTO teams (id, name, root_member_id, plan, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTeam :one
SELECT * FROM teams WHERE id = ?;

-- name: ListTeams :many
SELECT * FROM teams ORDER BY created_at;

-- name: UpdateTeam :execresult
UPDATE teams SET name = ?, root_member_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTeamPlan :execresult
UPDATE teams SET plan = ?, updated_at = ? WHERE id = ?;

-- name: DeleteTeam :execresult
DELETE FROM teams WHERE id = ?;

-- name: AddTeamMember :execresult
INSERT INTO team_members (team_id, member_id) VALUES (?, ?);

-- name: RemoveTeamMember :execresult
DELETE FROM team_members WHERE team_id = ? AND member_id = ?;

-- name: ListTeamMemberIDs :many
SELECT member_id FROM team_members WHERE team_id = ? ORDER BY rowid;

-- name: DeleteTeamMembers :execresult
DELETE FROM team_members WHERE team_id = ?;

-- name: AddTeamRelation :execresult
INSERT INTO team_relations (team_id, from_member_id, to_member_id, type) VALUES (?, ?, ?, ?);

-- name: RemoveTeamRelation :execresult
DELETE FROM team_relations WHERE team_id = ? AND from_member_id = ? AND to_member_id = ? AND type = ?;

-- name: ListTeamRelations :many
SELECT from_member_id, to_member_id, type FROM team_relations WHERE team_id = ? ORDER BY rowid;

-- name: RemoveTeamMemberRelations :execresult
DELETE FROM team_relations WHERE team_id = ? AND (from_member_id = ? OR to_member_id = ?);

-- name: DeleteTeamRelations :execresult
DELETE FROM team_relations WHERE team_id = ?;
