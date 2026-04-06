import type {
  DashboardData,
  TeamView,
  MemberView,
  MemberPlanView,
  ClaudeMdView,
  SkillView,
  SettingsView,
  ClaudeJsonView,
  GitRepoView,
  EnvView,
  TaskView,
  NoteView,
  MessageView,
} from "@/types";

declare global {
  interface Window {
    __CLIER_DATA__: DashboardData;
  }
}

function getData(): DashboardData {
  return window.__CLIER_DATA__;
}

function findById<T extends { id: string }>(items: T[], id: string): T | undefined {
  return items.find((item) => item.id === id);
}

export type {
  ClaudeMdView,
  SkillView,
  SettingsView,
  ClaudeJsonView,
  GitRepoView,
  MemberView,
  MemberPlanView,
  TeamView,
  EnvView,
  TaskView,
  NoteView,
  MessageView,
};

export const api = {
  claudeMds: {
    list: (): Promise<ClaudeMdView[]> => Promise.resolve(getData().claudeMds),
    get: (id: string): Promise<ClaudeMdView> => {
      const item = findById(getData().claudeMds, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  skills: {
    list: (): Promise<SkillView[]> => Promise.resolve(getData().skills),
    get: (id: string): Promise<SkillView> => {
      const item = findById(getData().skills, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  settings: {
    list: (): Promise<SettingsView[]> => Promise.resolve(getData().settings),
    get: (id: string): Promise<SettingsView> => {
      const item = findById(getData().settings, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  claudeJsons: {
    list: (): Promise<ClaudeJsonView[]> => Promise.resolve(getData().claudeJsons),
    get: (id: string): Promise<ClaudeJsonView> => {
      const item = findById(getData().claudeJsons, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  members: {
    list: (): Promise<MemberView[]> => Promise.resolve(getData().members),
    get: (id: string): Promise<MemberView> => {
      const item = findById(getData().members, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  teams: {
    list: (): Promise<TeamView[]> => Promise.resolve(getData().teams),
    get: (id: string): Promise<TeamView> => {
      const item = findById(getData().teams, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
    getStructure: (
      id: string,
    ): Promise<{
      rootTeamMemberId: string;
      members: Array<{
        id: string;
        memberId: string;
        name: string;
        model: string;
        skillIds: string[];
        skillNames: string[];
      }>;
      relations: Array<{ from: string; to: string }>;
    }> => {
      const data = getData();
      const team = findById(data.teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      const memberSpecMap = new Map(data.members.map((m) => [m.id, m]));
      const members = team.teamMembers.map((tm) => {
        const spec = memberSpecMap.get(tm.memberId);
        return {
          id: tm.id,
          memberId: tm.memberId,
          name: tm.name,
          model: spec?.model ?? "",
          skillIds: spec?.skillIds ?? [],
          skillNames: spec?.skillNames ?? [],
        };
      });
      return Promise.resolve({
        rootTeamMemberId: team.rootTeamMemberId,
        members,
        relations: team.relations,
      });
    },
    getMembers: (id: string): Promise<MemberView[]> => {
      const data = getData();
      const team = findById(data.teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      const memberSpecMap = new Map(data.members.map((m) => [m.id, m]));
      const memberIds = [...new Set(team.teamMembers.map((tm) => tm.memberId))];
      return Promise.resolve(memberIds.map((mid) => memberSpecMap.get(mid)).filter((m): m is MemberView => m != null));
    },
    getRelations: (id: string): Promise<Array<{ from: string; to: string }>> => {
      const team = findById(getData().teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      return Promise.resolve(team.relations);
    },
  },
  gitRepos: {
    list: (): Promise<GitRepoView[]> => Promise.resolve(getData().gitRepos),
    get: (id: string): Promise<GitRepoView> => {
      const item = findById(getData().gitRepos, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  envs: {
    list: (): Promise<EnvView[]> => Promise.resolve(getData().envs),
    get: (id: string): Promise<EnvView> => {
      const item = findById(getData().envs, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  tasks: {
    list: (): Promise<TaskView[]> => Promise.resolve(getData().tasks),
    get: (id: string): Promise<TaskView> => {
      const item = findById(getData().tasks, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
};
