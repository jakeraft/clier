import { useNavigate } from "react-router";
import type { Entity } from "@/lib/entities";
import { EntityTable, type EntityTableColumn, type EntityRow } from "@/components/entity-table";
import { ListLayout } from "@/components/list-layout";
import { useEntityList } from "@/hooks/use-entity-list";

interface EntityListPageProperties<T extends EntityRow> {
  entityType: Entity;
  apiList: (signal?: AbortSignal) => Promise<T[]>;
  columns: EntityTableColumn<T>[];
  empty: { title: string; description: string };
  routeBase?: string;
  renderName?: (item: T) => React.ReactNode;
  sortItems?: (items: T[]) => T[];
}

export function EntityListPage<T extends EntityRow>({
  entityType,
  apiList,
  columns,
  empty,
  routeBase,
  renderName,
  sortItems,
}: Readonly<EntityListPageProperties<T>>) {
  const navigate = useNavigate();
  const { items: rawItems, ready, error } = useEntityList<T>(apiList);

  const items = sortItems ? sortItems(rawItems) : rawItems;

  return (
    <ListLayout error={error}>
      <EntityTable
        entityType={entityType}
        items={items}
        ready={ready}
        columns={columns}
        empty={empty}
        renderName={renderName}
        onRowClick={routeBase ? (item) => void navigate(`${routeBase}/${item.id}`) : undefined}
      />
    </ListLayout>
  );
}
