import { api } from "@/api";
import type { MemberView } from "@/api";
import { EMPTY_DATA } from "@/components/empty-cell";
import { CountBadge } from "@/components/count-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<MemberView>[] = [
  {
    header: "Model",
    cell: (m) => m.model || EMPTY_DATA,
    flex: 2,
  },
  {
    header: "Git Repo",
    cell: (m) => m.gitRepoUrl || EMPTY_DATA,
    flex: 2,
  },
  {
    header: "Skills",
    cell: (m) => <CountBadge count={m.skillIds.length} />,
  },
];

export function Members() {
  return (
    <EntityListPage<MemberView>
      entityType="member"
      apiList={api.members.list}
      columns={columns}
      empty={{ title: "No members yet", description: "Members represent agents in teams" }}
      routeBase="/members"
    />
  );
}
