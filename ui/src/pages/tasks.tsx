import { api } from "@/api";
import type { TaskView } from "@/api";
import { EntityBadge } from "@/components/entity-badge";
import { StatusBadge } from "@/components/status-badge";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<TaskView>[] = [
  {
    header: "Name",
    cell: (s) => s.name,
    flex: 2,
  },
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

export function Tasks() {
  return (
    <EntityListPage<TaskView>
      entityType="task"
      apiList={api.tasks.list}
      columns={columns}
      empty={{ title: "No tasks yet", description: "Start a task with a team to get started" }}
      routeBase="/tasks"
    />
  );
}
