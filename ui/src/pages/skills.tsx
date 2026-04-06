import { api } from "@/api";
import type { SkillView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<SkillView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function Skills() {
  return (
    <EntityListPage<SkillView>
      entityType="skill"
      apiList={api.skills.list}
      columns={columns}
      empty={{ title: "No skills yet", description: "Claude Code skills for team members" }}
      routeBase="/skills"
    />
  );
}
