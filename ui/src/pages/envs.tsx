import { api } from "@/api";
import type { EnvView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<EnvView>[] = [
  { header: "Key", className: "max-w-xs truncate", cell: (c) => c.key },
  { header: "Value", className: "max-w-xs truncate", cell: (c) => c.value },
];

export function Envs() {
  return (
    <EntityListPage<EnvView>
      entityType="env"
      apiList={api.envs.list}
      columns={columns}
      empty={{ title: "No envs yet", description: "Create environment variables for agent isolation" }}
    />
  );
}
