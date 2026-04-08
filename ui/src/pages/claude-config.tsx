import { api } from "@/api";
import type { ClaudeSettingsView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const claudeSettingsColumns: EntityTableColumn<ClaudeSettingsView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function ClaudeConfig() {
  return (
    <EntityListPage<ClaudeSettingsView>
      entityType="claude-settings"
      apiList={api.claudeSettings.list}
      columns={claudeSettingsColumns}
      empty={{ title: "No settings.json yet", description: "Claude Code settings.json configurations" }}
      routeBase="/claude/claude-settings"
    />
  );
}
