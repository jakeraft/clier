import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import type { Entity } from "@/lib/entities";

interface EntityBadgeListProperties {
  entity: Entity;
  items: { id: string; name: string; to: string }[];
}

export function EntityBadgeList({ entity, items }: Readonly<EntityBadgeListProperties>) {
  if (items.length === 0) {
    return <EmptyEntityBadge entity={entity} />;
  }
  return (
    <span className={cn(flex.wrap, gap[1])}>
      {items.map((item) => (
        <EntityBadge key={item.id} to={item.to}>
          {item.name}
        </EntityBadge>
      ))}
    </span>
  );
}
