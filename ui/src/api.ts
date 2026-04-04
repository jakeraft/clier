import type {
  DashboardData,
  TeamView,
  MemberView,
  MemberPlanView,
  CliProfileView,
  SystemPromptView,
  GitRepoView,
  EnvView,
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
  CliProfileView,
  SystemPromptView,
  GitRepoView,
  MemberView,
  MemberPlanView,
  TeamView,
  EnvView,
};

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
    getStructure: (
      id: string,
    ): Promise<{
      rootTeamMemberId: string;
      members: Array<{
        id: string;
        memberId: string;
        name: string;
        cliProfileId: string;
        systemPromptIds: string[];
        cliProfileName: string;
        systemPromptNames: string[];
      }>;
      relations: Array<{ from: string; to: string; type: string }>;
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
          cliProfileId: spec?.cliProfileId ?? "",
          systemPromptIds: spec?.systemPromptIds ?? [],
          cliProfileName: spec?.cliProfileName ?? "",
          systemPromptNames: spec?.systemPromptNames ?? [],
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
    getRelations: (id: string): Promise<Array<{ from: string; to: string; type: string }>> => {
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
};
