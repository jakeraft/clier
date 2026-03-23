import { Link } from "react-router";
import { cn } from "@/lib/utilities";
import { ENTITY_STYLE, ENTITY_ICON, entityFromPath } from "@/lib/entities";
import { Badge } from "@/components/ui/badge";

interface EntityBadgeProperties {
  to: string;
  children: React.ReactNode;
  className?: string;
}

export function EntityBadge({ to, children, className }: Readonly<EntityBadgeProperties>) {
  const entity = entityFromPath(to);
  const Icon = entity ? ENTITY_ICON.get(entity) : undefined;
  return (
    <Badge asChild className={cn(entity ? ENTITY_STYLE.get(entity) : "hover:bg-accent", "cursor-pointer", className)}>
      <Link to={to} onClick={(event) => event.stopPropagation()}>
        {Icon ? <Icon /> : undefined}
        {children}
      </Link>
    </Badge>
  );
}
