import { api } from "@/api";
import type { CliProfileView } from "@/api";
import { ArgumentsBadge } from "@/components/arguments-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<CliProfileView>[] = [
  { header: "Model", cell: (a) => a.model, flex: 2 },
  { header: "Binary", cell: (a) => a.binary, flex: 1 },
  {
    header: "System",
    cell: (a) => <ArgumentsBadge args={a.systemArgs} variant="secondary" compact side="right" />,
    flex: 1,
  },
  {
    header: "Custom",
    cell: (a) => (
      <ArgumentsBadge
        args={a.customArgs.join(" ").split(/\s+/).filter(Boolean)}
        variant="outline"
        compact
        side="right"
      />
    ),
    flex: 1,
  },
];

export function CliProfiles() {
  return (
    <EntityListPage<CliProfileView>
      entityType="cli-profile"
      apiList={api.cliProfiles.list}
      columns={columns}
      empty={{ title: "No CLI profiles yet", description: "CLI profile configurations for teams" }}
      routeBase="/cli-profiles"
    />
  );
}
