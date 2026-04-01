// --- API types (auto-generated from OpenAPI spec) ---
// Run `pnpm generate:api` to regenerate api-types.ts from the backend's OpenAPI spec.

import type { operations } from "@/lib/api-types";

type ResponseBody<T extends keyof operations> =
  operations[T] extends { responses: { 200: { content: { "application/json": infer R } } } } ? R :
  operations[T] extends { responses: { 201: { content: { "application/json": infer R } } } } ? R :
  never;

export type CliProfileView = ResponseBody<"listCliProfiles">[number];
export type SystemPromptView = ResponseBody<"listSystemPrompts">[number];
export type EnvironmentView = ResponseBody<"listEnvironments">[number];
export type GitRepoView = ResponseBody<"listGitRepos">[number];
export type MemberView = ResponseBody<"listMembers">[number];
export type TeamView = ResponseBody<"listTeams">[number];
export type SprintView = ResponseBody<"listSprints">[number];
export type BrandInfo = ResponseBody<"getSystemInfo">;

// --- Request layer --

interface ProblemDetail {
  type?: string;
  title?: string;
  status?: number;
  detail?: string;
}

interface StructureResponse {
  rootMemberId: string;
  members: MemberView[];
  relations: Array<{ from: string; to: string; type: string }>;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: HeadersInit = options?.body ? { "Content-Type": "application/json" } : {};
  const response = await fetch(`/api${path}`, { headers, ...options });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ProblemDetail;
    throw new Error(body.detail ?? body.title ?? `${response.status} ${response.statusText}`);
  }
  if (response.status === 204) return undefined as T;
  try {
    return (await response.json()) as T;
  } catch {
    throw new Error("Invalid response from server");
  }
}

function get<T>(path: string, signal?: AbortSignal) {
  return request<T>(path, signal ? { signal } : undefined);
}

// --- API client --

export const api = {
  system: {
    info: (signal?: AbortSignal) => get<BrandInfo>("/system/info", signal),
  },
  cliProfiles: {
    list: (signal?: AbortSignal) => get<CliProfileView[]>("/cli-profile", signal),
    get: (id: string, signal?: AbortSignal) => get<CliProfileView>(`/cli-profile/${id}`, signal),
  },
  environments: {
    list: (signal?: AbortSignal) => get<EnvironmentView[]>("/env", signal),
    get: (id: string, signal?: AbortSignal) => get<EnvironmentView>(`/env/${id}`, signal),
  },
  systemPrompts: {
    list: (signal?: AbortSignal) => get<SystemPromptView[]>("/system-prompt", signal),
    get: (id: string, signal?: AbortSignal) => get<SystemPromptView>(`/system-prompt/${id}`, signal),
  },
  members: {
    list: (signal?: AbortSignal) => get<MemberView[]>("/member", signal),
    get: (id: string, signal?: AbortSignal) => get<MemberView>(`/member/${id}`, signal),
  },
  teams: {
    list: (signal?: AbortSignal) => get<TeamView[]>("/team", signal),
    get: (id: string, signal?: AbortSignal) => get<TeamView>(`/team/${id}`, signal),
    getStructure: (id: string, signal?: AbortSignal) => get<StructureResponse>(`/team/${id}/structure`, signal),
    getMembers: (id: string, signal?: AbortSignal) => get<MemberView[]>(`/team/${id}/members`, signal),
    getRelations: (id: string, signal?: AbortSignal) =>
      get<Array<{ from: string; to: string; type: string }>>(`/team/${id}/relations`, signal),
  },
  sprints: {
    list: (signal?: AbortSignal) => get<SprintView[]>("/sprint", signal),
    get: (id: string, signal?: AbortSignal) => get<SprintView>(`/sprint/${id}`, signal),
  },
  gitRepos: {
    list: (signal?: AbortSignal) => get<GitRepoView[]>("/git-repo", signal),
    get: (id: string, signal?: AbortSignal) => get<GitRepoView>(`/git-repo/${id}`, signal),
  },
};
