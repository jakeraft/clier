import { api } from "@/api";
import type { MemberView } from "@/api";
import { EMPTY_DATA } from "@/components/empty-cell";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import { CountBadge } from "@/components/count-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<MemberView>[] = [
  {
    header: "CLI Profile",
    cell: (m) => <EntityBadge to={`/cli-profiles/${m.cliProfileId}`}>{m.cliProfileName || EMPTY_DATA}</EntityBadge>,
    flex: 2,
  },
  {
    header: "Git Repo",
    cell: (m) =>
      m.gitRepoId ? (
        <EntityBadge to="/git-repos">{m.gitRepoName || EMPTY_DATA}</EntityBadge>
      ) : (
        <EmptyEntityBadge entity="git-repo" />
      ),
    flex: 2,
  },
  {
    header: "System Prompt",
    cell: (m) => <CountBadge count={m.systemPromptIds.length} />,
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
