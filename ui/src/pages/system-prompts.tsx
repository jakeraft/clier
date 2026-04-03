import { api } from "@/api";
import type { SystemPromptView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SystemPromptView>[] = [
  { header: "Prompt", cell: (c) => c.prompt, flex: 2 },
];

export function SystemPrompts() {
  return (
    <EntityListPage<SystemPromptView>
      entityType="system-prompt"
      apiList={api.systemPrompts.list}
      columns={columns}
      empty={{ title: "No system prompts yet", description: "Create system prompts to guide team behavior" }}
      routeBase="/system-prompts"
    />
  );
}
