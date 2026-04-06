CREATE TABLE IF NOT EXISTS cli_profiles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    model       TEXT NOT NULL,
    binary      TEXT NOT NULL DEFAULT 'claude',
    system_args TEXT NOT NULL DEFAULT '[]',
    custom_args TEXT NOT NULL DEFAULT '[]',
    settings_json TEXT NOT NULL DEFAULT '{}',
    claude_json   TEXT NOT NULL DEFAULT '{}',
    created_at    INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS system_prompts (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    prompt     TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS git_repos (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    cli_profile_id TEXT NOT NULL REFERENCES cli_profiles(id) ON DELETE RESTRICT,
    git_repo_id    TEXT REFERENCES git_repos(id) ON DELETE RESTRICT,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS teams (
    id                   TEXT PRIMARY KEY,
    name                 TEXT NOT NULL,
    -- References team_members(id). FK intentionally omitted: circular dependency
    -- (team must exist before team_member, but root requires team_member).
    -- Invariant enforced at domain layer (domain/team.go).
    root_team_member_id  TEXT NOT NULL,
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL
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

CREATE TABLE IF NOT EXISTS terminal_refs (
    task_id        TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id TEXT NOT NULL,
    refs           TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (task_id, team_member_id)
);

CREATE TABLE IF NOT EXISTS member_system_prompts (
    member_id        TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    system_prompt_id TEXT NOT NULL REFERENCES system_prompts(id) ON DELETE RESTRICT,
    PRIMARY KEY (member_id, system_prompt_id)
);

CREATE TABLE IF NOT EXISTS envs (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS member_envs (
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    env_id    TEXT NOT NULL REFERENCES envs(id)    ON DELETE RESTRICT,
    PRIMARY KEY (member_id, env_id)
);
