import { api } from "@/api";
import type { TeamView } from "@/api";
import { EMPTY_DATA } from "@/components/empty-cell";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import { CountBadge } from "@/components/count-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<TeamView>[] = [
  {
    header: "Root",
    cell: (t) =>
      t.rootMemberId ? (
        <EntityBadge to={`/members/${t.rootMemberId}`}>{t.rootMemberName || EMPTY_DATA}</EntityBadge>
      ) : (
        <EmptyEntityBadge entity="member" />
      ),
    flex: 2,
  },
  {
    header: "Member",
    cell: (t) => <CountBadge count={t.memberIds.length} />,
    flex: 1,
  },
];

export function Teams() {
  return (
    <EntityListPage<TeamView>
      entityType="team"
      apiList={api.teams.list}
      columns={columns}
      empty={{ title: "No teams yet", description: "Create a team to get started" }}
      routeBase="/teams"
    />
  );
}
