export interface DashboardData {
  teams: TeamView[];
  members: MemberView[];
  agentDotMds: AgentDotMdView[];
  skills: SkillView[];
  claudeSettings: ClaudeSettingsView[];
  claudeJsons: ClaudeJsonView[];
  gitRepos: GitRepoView[];
  tasks: TaskView[];
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

export interface TaskView {
  id: string;
  name: string;
  teamId: string;
  teamName: string;
  status: string;
  plan: MemberPlanView[];
  notes: NoteView[];
  messages: MessageView[];
  createdAt: string;
  updatedAt: string;
}

export interface NoteView {
  id: string;
  teamMemberId: string;
  memberName: string;
  content: string;
  createdAt: string;
}

export interface MessageView {
  id: string;
  fromTeamMemberId: string;
  fromMemberName: string;
  toTeamMemberId: string;
  toMemberName: string;
  content: string;
  createdAt: string;
}

export interface RelationView {
  from: string;
  to: string;
}

export interface MemberView {
  id: string;
  name: string;
  agentType: string;
  model: string;
  args: string[];
  agentDotMdId: string | null;
  skillIds: string[];
  claudeSettingsId: string | null;
  claudeJsonId: string | null;
  gitRepoId: string | null;
  agentDotMdName: string | null;
  skillNames: string[];
  claudeSettingsName: string | null;
  claudeJsonName: string | null;
  gitRepoName: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AgentDotMdView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface SkillView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClaudeSettingsView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClaudeJsonView {
  id: string;
  name: string;
  content: string;
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

