import { api } from "@/api";
import type { ClaudeMdView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<ClaudeMdView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function ClaudeMds() {
  return (
    <EntityListPage<ClaudeMdView>
      entityType="claude-md"
      apiList={api.claudeMds.list}
      columns={columns}
      empty={{ title: "No CLAUDE.md files yet", description: "CLAUDE.md project instructions for team members" }}
      routeBase="/claude-mds"
    />
  );
}
