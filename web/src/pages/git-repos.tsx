import { api } from "@/api";
import type { GitRepoView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<GitRepoView>[] = [
  {
    header: "URL",
    cell: (w) => {
      const href = w.url.startsWith("http") ? w.url : `https://${w.url}`;
      return (
        <a href={href} target="_blank" rel="noopener noreferrer">
          {w.url}
        </a>
      );
    },
  },
];

export function GitRepos() {
  return (
    <EntityListPage<GitRepoView>
      entityType="git-repo"
      apiList={api.gitRepos.list}
      columns={columns}
      empty={{ title: "No git repos yet", description: "Create git repos for members" }}
    />
  );
}
