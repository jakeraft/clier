import { api } from "@/api";
import type { SessionView } from "@/api";
import { EntityBadge } from "@/components/entity-badge";
import { StatusBadge } from "@/components/status-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SessionView>[] = [
  {
    header: "Status",
    cell: (s) => <StatusBadge status={s.status} />,
    flex: 1,
  },
  {
    header: "Team",
    cell: (s) => <EntityBadge to={`/teams/${s.teamId}`}>{s.teamName}</EntityBadge>,
    flex: 2,
  },
];

export function Sessions() {
  return (
    <EntityListPage<SessionView>
      entityType="session"
      apiList={api.sessions.list}
      columns={columns}
      empty={{ title: "No sessions yet", description: "Start a session with a team to get started" }}
      routeBase="/sessions"
    />
  );
}
