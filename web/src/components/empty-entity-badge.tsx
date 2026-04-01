import type { Entity } from "@/lib/entities";
import { ENTITY_STYLE } from "@/lib/entities";
import { EMPTY_DATA } from "@/components/empty-cell";
import { Badge } from "@/components/ui/badge";

interface EmptyEntityBadgeProperties {
  entity: Entity;
}

/** Styled placeholder badge for missing entity references. */
export function EmptyEntityBadge({ entity }: Readonly<EmptyEntityBadgeProperties>) {
  return <Badge className={ENTITY_STYLE.get(entity)}>{EMPTY_DATA}</Badge>;
}
