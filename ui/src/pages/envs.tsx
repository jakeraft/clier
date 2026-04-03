import { api } from "@/api";
import type { EnvView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

function maskValue(value: string): string {
  if (value.length <= 3) return "***";
  return value.slice(0, 3) + "***";
}

const columns: EntityTableColumn<EnvView>[] = [
  { header: "Key", cell: (c) => c.key, flex: 1 },
  { header: "Value", cell: (c) => maskValue(c.value), flex: 2 },
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
