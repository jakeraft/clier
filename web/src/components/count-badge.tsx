import { Badge } from "@/components/ui/badge";
import { EMPTY_DATA } from "@/components/empty-cell";

interface CountBadgeProperties {
  count: number;
}

export function CountBadge({ count }: Readonly<CountBadgeProperties>) {
  if (count === 0) return <Badge variant="outline">{EMPTY_DATA}</Badge>;

  return <Badge variant="outline">+{count}</Badge>;
}
