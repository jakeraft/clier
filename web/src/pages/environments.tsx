import { api } from "@/api";
import type { EnvironmentView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const MASK_VISIBLE_PREFIX = 2;
const MASK_MAX_STARS = 8;

/** Show the first 2 characters then mask the rest (up to 8 stars). Short values are fully masked. */
function maskValue(value: string): string {
  if (value.length === 0) return value;
  if (value.length <= 3) return "*".repeat(value.length);
  return `${value.slice(0, MASK_VISIBLE_PREFIX)}${"*".repeat(Math.min(value.length - MASK_VISIBLE_PREFIX, MASK_MAX_STARS))}`;
}

const columns: EntityTableColumn<EnvironmentView>[] = [
  { header: "Key", cell: (environment) => environment.key },
  { header: "Value", cell: (environment) => maskValue(environment.value) },
];

export function Environments() {
  return (
    <EntityListPage<EnvironmentView>
      entityType="environment"
      apiList={api.environments.list}
      columns={columns}
      empty={{ title: "No environments yet", description: "Create environment variables for members" }}
    />
  );
}
