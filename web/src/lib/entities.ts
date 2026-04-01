import {
  Users,
  User,
  Bot,
  BookOpen,
  KeyRound,
  FolderGit2,
  History,
  type LucideIcon,
} from "lucide-react";
type Entity = "team" | "member" | "cli-profile" | "system-prompt" | "environment" | "git-repo" | "sprint";

const ENTITY_STYLE = new Map<Entity, string>([
  ["team", "bg-entity-team/10 text-entity-team hover:bg-entity-team/20 [a&]:hover:bg-entity-team/20"],
  ["member", "bg-entity-member/10 text-entity-member hover:bg-entity-member/20 [a&]:hover:bg-entity-member/20"],
  ["cli-profile", "bg-entity-model/10 text-entity-model hover:bg-entity-model/20 [a&]:hover:bg-entity-model/20"],
  [
    "system-prompt",
    "bg-entity-instruction/10 text-entity-instruction hover:bg-entity-instruction/20 [a&]:hover:bg-entity-instruction/20",
  ],
  [
    "environment",
    "bg-entity-environment/10 text-entity-environment hover:bg-entity-environment/20 [a&]:hover:bg-entity-environment/20",
  ],
  [
    "git-repo",
    "bg-entity-workspace/10 text-entity-workspace hover:bg-entity-workspace/20 [a&]:hover:bg-entity-workspace/20",
  ],
  ["sprint", "bg-entity-sprint/10 text-entity-sprint hover:bg-entity-sprint/20 [a&]:hover:bg-entity-sprint/20"],
]);

const ENTITY_ICON = new Map<Entity, LucideIcon>([
  ["team", Users],
  ["member", User],
  ["cli-profile", Bot],
  ["system-prompt", BookOpen],
  ["environment", KeyRound],
  ["git-repo", FolderGit2],
  ["sprint", History],
]);

const SEGMENT_TO_ENTITY = new Map<string, Entity>([
  ["teams", "team"],
  ["members", "member"],
  ["cli-profiles", "cli-profile"],
  ["system-prompts", "system-prompt"],
  ["environments", "environment"],
  ["git-repos", "git-repo"],
  ["sprints", "sprint"],
]);

function entityFromPath(path: string): Entity | undefined {
  const match = /^\/(teams|members|cli-profiles|system-prompts|environments|git-repos|sprints)(\/|$)/.exec(path);
  if (!match) return undefined;
  return SEGMENT_TO_ENTITY.get(match[1]);
}

export { ENTITY_STYLE, ENTITY_ICON, entityFromPath, type Entity };
