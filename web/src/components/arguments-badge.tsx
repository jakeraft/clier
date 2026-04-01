import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { Badge } from "@/components/ui/badge";
import { CountBadge } from "@/components/count-badge";
import { EMPTY_DATA } from "@/components/empty-cell";

interface ArgumentsBadgeProperties {
  args: string[];
  variant?: "secondary" | "outline";
  /** Compact mode: shows CountBadge (+N) with hover popover (for list views) */
  compact?: boolean;
  /** Popover direction in compact mode */
  side?: "top" | "right" | "bottom" | "left";
}

export function ArgumentsBadge({
  args,
  variant = "secondary",
  compact,
  side,
}: Readonly<ArgumentsBadgeProperties>): React.ReactNode {
  const badges = (
    <div className={cn(flex.wrap, gap[1])}>
      {args.map((a, index) => (
        // eslint-disable-next-line @eslint-react/no-array-index-key -- args are plain strings with possible duplicates; no stable unique id available
        <Badge key={`${a}-${index}`} variant={variant}>
          {a}
        </Badge>
      ))}
    </div>
  );

  if (args.length === 0) {
    return <Badge variant="outline">{EMPTY_DATA}</Badge>;
  }

  if (compact) {
    return <CountBadge count={args.length} />;
  }

  return badges;
}
