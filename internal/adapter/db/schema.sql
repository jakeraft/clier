CREATE TABLE IF NOT EXISTS cli_profiles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    model       TEXT NOT NULL,
    binary      TEXT NOT NULL DEFAULT 'claude',
    system_args TEXT NOT NULL DEFAULT '[]',
    custom_args TEXT NOT NULL DEFAULT '[]',
    dot_config  TEXT NOT NULL DEFAULT '{}',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS system_prompts (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    prompt     TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS environments (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
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
    cli_profile_id TEXT NOT NULL REFERENCES cli_profiles(id),
    git_repo_id    TEXT REFERENCES git_repos(id),
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS teams (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    root_member_id TEXT NOT NULL REFERENCES members(id),
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sprints (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    team_snapshot TEXT NOT NULL DEFAULT '{}',
    state         TEXT NOT NULL DEFAULT 'running',
    error         TEXT NOT NULL DEFAULT '',
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id             TEXT PRIMARY KEY,
    sprint_id      TEXT NOT NULL REFERENCES sprints(id),
    from_member_id TEXT NOT NULL,
    to_member_id   TEXT NOT NULL,
    content        TEXT NOT NULL,
    created_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id   TEXT NOT NULL REFERENCES teams(id),
    member_id TEXT NOT NULL REFERENCES members(id),
    PRIMARY KEY (team_id, member_id)
);

CREATE TABLE IF NOT EXISTS team_relations (
    team_id        TEXT NOT NULL REFERENCES teams(id),
    from_member_id TEXT NOT NULL REFERENCES members(id),
    to_member_id   TEXT NOT NULL REFERENCES members(id),
    type           TEXT NOT NULL,
    PRIMARY KEY (team_id, from_member_id, to_member_id, type)
);

CREATE TABLE IF NOT EXISTS member_system_prompts (
    member_id        TEXT NOT NULL REFERENCES members(id),
    system_prompt_id TEXT NOT NULL REFERENCES system_prompts(id),
    PRIMARY KEY (member_id, system_prompt_id)
);

CREATE TABLE IF NOT EXISTS member_environments (
    member_id      TEXT NOT NULL REFERENCES members(id),
    environment_id TEXT NOT NULL REFERENCES environments(id),
    PRIMARY KEY (member_id, environment_id)
);
