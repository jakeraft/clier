CREATE TABLE IF NOT EXISTS claude_mds (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS claude_settings (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS claude_jsons (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    agent_type         TEXT NOT NULL DEFAULT 'claude',
    model              TEXT NOT NULL,
    args               TEXT NOT NULL DEFAULT '[]',
    claude_md_id    TEXT REFERENCES claude_mds(id) ON DELETE RESTRICT,
    claude_settings_id TEXT REFERENCES claude_settings(id) ON DELETE RESTRICT,
    claude_json_id     TEXT REFERENCES claude_jsons(id) ON DELETE RESTRICT,
    git_repo_url       TEXT NOT NULL DEFAULT '',
    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS member_skills (
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    skill_id  TEXT NOT NULL REFERENCES skills(id)  ON DELETE RESTRICT,
    PRIMARY KEY (member_id, skill_id)
);

CREATE TABLE IF NOT EXISTS teams (
    id                   TEXT PRIMARY KEY,
    name                 TEXT NOT NULL,
    root_team_member_id  TEXT NOT NULL,
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS team_members (
    id        TEXT PRIMARY KEY,
    team_id   TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    name      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS team_relations (
    team_id              TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    from_team_member_id  TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    to_team_member_id    TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, from_team_member_id, to_team_member_id)
);

CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL DEFAULT '',
    team_id       TEXT NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    status        TEXT NOT NULL DEFAULT 'running',
    plan          TEXT NOT NULL DEFAULT '[]',
    created_at    INTEGER NOT NULL,
    stopped_at    INTEGER
);

CREATE TABLE IF NOT EXISTS messages (
    id                    TEXT PRIMARY KEY,
    task_id               TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    from_team_member_id   TEXT,
    to_team_member_id     TEXT NOT NULL,
    content               TEXT NOT NULL,
    created_at            INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
    id              TEXT PRIMARY KEY,
    task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id  TEXT NOT NULL,
    content         TEXT NOT NULL,
    created_at      INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS terminal_refs (
    task_id        TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id TEXT NOT NULL,
    refs           TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (task_id, team_member_id)
);
