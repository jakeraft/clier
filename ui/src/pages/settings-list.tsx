import { api } from "@/api";
import type { SettingsView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SettingsView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function SettingsList() {
  return (
    <EntityListPage<SettingsView>
      entityType="claude-settings"
      apiList={api.settings.list}
      columns={columns}
      empty={{ title: "No settings yet", description: "Claude Code settings.json configurations" }}
      routeBase="/claude-settings"
    />
  );
}
