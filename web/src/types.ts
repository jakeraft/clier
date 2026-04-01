export interface DashboardData {
  teams: TeamView[];
  members: MemberView[];
  sprints: SprintView[];
  cliProfiles: CliProfileView[];
  systemPrompts: SystemPromptView[];
  gitRepos: GitRepoView[];
}

export interface TeamView {
  id: string;
  name: string;
  rootMemberId: string;
  memberIds: string[];
  relations: RelationView[];
  rootMemberName: string;
  memberNames: string[];
  createdAt: string;
  updatedAt: string;
}

export interface RelationView {
  from: string;
  to: string;
  type: "leader" | "peer";
}

export interface MemberView {
  id: string;
  name: string;
  cliProfileId: string;
  systemPromptIds: string[];
  gitRepoId: string | null;
  cliProfileName: string;
  systemPromptNames: string[];
  gitRepoName: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface SprintView {
  id: string;
  name: string;
  teamSnapshot: TeamSnapshotView;
  state: "running" | "completed" | "errored";
  error: string | null;
  teamName: string;
  createdAt: string;
  updatedAt: string;
}

export interface TeamSnapshotView {
  teamName: string;
  rootMemberId: string;
  members: MemberSnapshotView[];
}

export interface MemberSnapshotView {
  memberId: string;
  memberName: string;
  binary: "claude" | "codex";
  model: string;
  cliProfileName: string;
  systemArgs: string[];
  customArgs: string[];
  dotConfig: Record<string, unknown>;
  systemPrompts: { name: string; prompt: string }[];
  gitRepo: { name: string; url: string } | null;
  relations: {
    leaders: string[];
    workers: string[];
    peers: string[];
  };
  protocol: string;
}

export interface CliProfileView {
  id: string;
  name: string;
  model: string;
  binary: "claude" | "codex";
  systemArgs: string[];
  customArgs: string[];
  dotConfig: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface SystemPromptView {
  id: string;
  name: string;
  prompt: string;
  createdAt: string;
  updatedAt: string;
}

export interface GitRepoView {
  id: string;
  name: string;
  url: string;
  createdAt: string;
  updatedAt: string;
}
