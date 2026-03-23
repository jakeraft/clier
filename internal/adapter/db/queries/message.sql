-- name: CreateMessage :exec
INSERT INTO messages (id, sprint_id, from_member_id, to_member_id, content, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMessagesBySprintID :many
SELECT * FROM messages WHERE sprint_id = ? ORDER BY created_at;

-- name: ListMessagesBySprintAndMember :many
SELECT * FROM messages
WHERE sprint_id = ? AND (from_member_id = ? OR to_member_id = ?)
ORDER BY created_at;
