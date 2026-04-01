import { Lock } from "lucide-react";
import { api } from "@/api";
import type { SystemPromptView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SystemPromptView>[] = [
  { header: "Prompt", className: "max-w-md truncate", cell: (c) => c.prompt },
];

export function SystemPrompts() {
  return (
    <EntityListPage<SystemPromptView>
      entityType="system-prompt"
      apiList={api.systemPrompts.list}
      columns={columns}
      empty={{ title: "No system prompts yet", description: "Create system prompts to guide team behavior" }}
      routeBase="/system-prompts"
      renderName={(c) => (
        <span className="flex items-center gap-1.5">
          {c.builtIn && <Lock className="size-3.5 shrink-0" />}
          {c.name}
        </span>
      )}
    />
  );
}
