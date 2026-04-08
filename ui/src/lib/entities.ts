import { Users, User, FileText, BookOpen, Settings2, FolderGit2, Play, type LucideIcon } from "lucide-react";
type Entity = "team" | "task" | "member" | "claude-md" | "skill" | "claude-settings" | "git-repo";

const ENTITY_STYLE = new Map<Entity, string>([
  ["team", "bg-entity-team/10 text-entity-team hover:bg-entity-team/20 [a&]:hover:bg-entity-team/20"],
  ["task", "bg-entity-task/10 text-entity-task hover:bg-entity-task/20 [a&]:hover:bg-entity-task/20"],
  ["member", "bg-entity-member/10 text-entity-member hover:bg-entity-member/20 [a&]:hover:bg-entity-member/20"],
  ["claude-md", "bg-entity-instruction/10 text-entity-instruction hover:bg-entity-instruction/20 [a&]:hover:bg-entity-instruction/20"],
  ["skill", "bg-entity-instruction/10 text-entity-instruction hover:bg-entity-instruction/20 [a&]:hover:bg-entity-instruction/20"],
  ["claude-settings", "bg-entity-model/10 text-entity-model hover:bg-entity-model/20 [a&]:hover:bg-entity-model/20"],
  [
    "git-repo",
    "bg-entity-workspace/10 text-entity-workspace hover:bg-entity-workspace/20 [a&]:hover:bg-entity-workspace/20",
  ],
]);

const ENTITY_ICON = new Map<Entity, LucideIcon>([
  ["team", Users],
  ["task", Play],
  ["member", User],
  ["claude-md", FileText],
  ["skill", BookOpen],
  ["claude-settings", Settings2],
  ["git-repo", FolderGit2],
]);

const SEGMENT_TO_ENTITY = new Map<string, Entity>([
  ["teams", "team"],
  ["tasks", "task"],
  ["members", "member"],
  ["prompts", "claude-md"],
  ["claude", "claude-settings"],
  ["claude-mds", "claude-md"],
  ["skills", "skill"],
  ["claude-settings", "claude-settings"],
  ["git-repos", "git-repo"],
]);

function entityFromPath(path: string): Entity | undefined {
  const match = /^\/(teams|tasks|members|prompts|claude|claude-mds|skills|claude-settings|git-repos)(\/|$)/.exec(path);
  if (!match) return undefined;
  return SEGMENT_TO_ENTITY.get(match[1]);
}

export { ENTITY_STYLE, ENTITY_ICON, entityFromPath, type Entity };
