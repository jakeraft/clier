import type {
  DashboardData,
  TeamView,
  MemberView,
  SprintView,
  CliProfileView,
  SystemPromptView,
  GitRepoView,
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

export type { CliProfileView, SystemPromptView, GitRepoView, MemberView, TeamView, SprintView };

export const api = {
  cliProfiles: {
    list: (): Promise<CliProfileView[]> => Promise.resolve(getData().cliProfiles),
    get: (id: string): Promise<CliProfileView> => {
      const item = findById(getData().cliProfiles, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  systemPrompts: {
    list: (): Promise<SystemPromptView[]> => Promise.resolve(getData().systemPrompts),
    get: (id: string): Promise<SystemPromptView> => {
      const item = findById(getData().systemPrompts, id);
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
    getStructure: (id: string): Promise<{
      rootMemberId: string;
      members: Array<MemberView & { cliProfileName: string; systemPromptNames: string[] }>;
      relations: Array<{ from: string; to: string; type: string }>;
    }> => {
      const data = getData();
      const team = findById(data.teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      const teamMembers = data.members.filter((m) => team.memberIds.includes(m.id));
      return Promise.resolve({
        rootMemberId: team.rootMemberId,
        members: teamMembers,
        relations: team.relations,
      });
    },
    getMembers: (id: string): Promise<MemberView[]> => {
      const data = getData();
      const team = findById(data.teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      return Promise.resolve(data.members.filter((m) => team.memberIds.includes(m.id)));
    },
    getRelations: (id: string): Promise<Array<{ from: string; to: string; type: string }>> => {
      const team = findById(getData().teams, id);
      if (!team) return Promise.reject(new Error("Not found"));
      return Promise.resolve(team.relations);
    },
  },
  sprints: {
    list: (): Promise<SprintView[]> => Promise.resolve(getData().sprints),
    get: (id: string): Promise<SprintView> => {
      const item = findById(getData().sprints, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
  gitRepos: {
    list: (): Promise<GitRepoView[]> => Promise.resolve(getData().gitRepos),
    get: (id: string): Promise<GitRepoView> => {
      const item = findById(getData().gitRepos, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
  },
};
