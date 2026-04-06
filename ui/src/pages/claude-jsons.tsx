import { api } from "@/api";
import type { ClaudeJsonView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<ClaudeJsonView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function ClaudeJsons() {
  return (
    <EntityListPage<ClaudeJsonView>
      entityType="claude-json"
      apiList={api.claudeJsons.list}
      columns={columns}
      empty={{ title: "No .claude.json files yet", description: "Claude Code .claude.json project configurations" }}
      routeBase="/claude-jsons"
    />
  );
}
