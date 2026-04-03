import { Users, User, Bot, BookOpen, FolderGit2, KeyRound, type LucideIcon } from "lucide-react";
type Entity = "team" | "member" | "cli-profile" | "system-prompt" | "git-repo" | "env";

const ENTITY_STYLE = new Map<Entity, string>([
  ["team", "bg-entity-team/10 text-entity-team hover:bg-entity-team/20 [a&]:hover:bg-entity-team/20"],
  ["member", "bg-entity-member/10 text-entity-member hover:bg-entity-member/20 [a&]:hover:bg-entity-member/20"],
  ["cli-profile", "bg-entity-model/10 text-entity-model hover:bg-entity-model/20 [a&]:hover:bg-entity-model/20"],
  [
    "system-prompt",
    "bg-entity-instruction/10 text-entity-instruction hover:bg-entity-instruction/20 [a&]:hover:bg-entity-instruction/20",
  ],
  [
    "git-repo",
    "bg-entity-workspace/10 text-entity-workspace hover:bg-entity-workspace/20 [a&]:hover:bg-entity-workspace/20",
  ],
  ["env", "bg-entity-env/10 text-entity-env hover:bg-entity-env/20 [a&]:hover:bg-entity-env/20"],
]);

const ENTITY_ICON = new Map<Entity, LucideIcon>([
  ["team", Users],
  ["member", User],
  ["cli-profile", Bot],
  ["system-prompt", BookOpen],
  ["git-repo", FolderGit2],
  ["env", KeyRound],
]);

const SEGMENT_TO_ENTITY = new Map<string, Entity>([
  ["teams", "team"],
  ["members", "member"],
  ["cli-profiles", "cli-profile"],
  ["system-prompts", "system-prompt"],
  ["git-repos", "git-repo"],
  ["envs", "env"],
]);

function entityFromPath(path: string): Entity | undefined {
  const match = /^\/(teams|members|cli-profiles|system-prompts|git-repos|envs)(\/|$)/.exec(path);
  if (!match) return undefined;
  return SEGMENT_TO_ENTITY.get(match[1]);
}

export { ENTITY_STYLE, ENTITY_ICON, entityFromPath, type Entity };
