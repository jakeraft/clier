import { api } from "@/api";
import type { SprintView } from "@/api";
import { StateBadge } from "@/components/state-badge";
import { EntityBadge } from "@/components/entity-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SprintView>[] = [
  {
    header: "State",
    cell: (r) => <StateBadge state={r.state} className="min-w-20" />,
  },
  {
    header: "Team",
    cell: (r) => <EntityBadge to="/teams">{r.teamName}</EntityBadge>,
  },
];

export function Sprints() {
  return (
    <EntityListPage<SprintView>
      entityType="sprint"
      apiList={api.sprints.list}
      columns={columns}
      empty={{ title: "No active sprints", description: "Run a team to see active executions here" }}
      routeBase="/sprints"
    />
  );
}
