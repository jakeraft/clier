export interface DashboardData {
  teams: TeamView[];
  members: MemberView[];
  cliProfiles: CliProfileView[];
  systemPrompts: SystemPromptView[];
  gitRepos: GitRepoView[];
  envs: EnvView[];
}

export interface TeamMemberView {
  id: string;
  memberId: string;
  name: string;
}

export interface TeamView {
  id: string;
  name: string;
  rootTeamMemberId: string;
  teamMemberIds: string[];
  teamMembers: TeamMemberView[];
  relations: RelationView[];
  plan: MemberPlanView[];
  rootMemberName: string;
  memberNames: string[];
  createdAt: string;
  updatedAt: string;
}

export interface MemberPlanView {
  teamMemberId: string;
  memberName: string;
  memberspace: string;
  command: string;
  gitRepo: { name: string; url: string } | null;
  files: { path: string; content: string }[];
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
  envIds: string[];
  gitRepoId: string | null;
  cliProfileName: string;
  systemPromptNames: string[];
  envNames: string[];
  gitRepoName: string | null;
  createdAt: string;
  updatedAt: string;
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

export interface EnvView {
  id: string;
  name: string;
  key: string;
  value: string;
  createdAt: string;
  updatedAt: string;
}

