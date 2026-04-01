import { cn } from "@/lib/utilities";
import { STATE_STYLE, STATE_ICON, type State } from "@/lib/states";
import { Badge } from "@/components/ui/badge";

// All badges share the same width so switching states doesn't cause layout shift
const MIN_WIDTH = Math.max(...[...STATE_STYLE.keys()].map((s) => s.length)) + 2;

interface StateBadgeProperties {
  state: State;
  className?: string;
}

export function StateBadge({ state, className }: Readonly<StateBadgeProperties>) {
  const Icon = STATE_ICON.get(state);
  return (
    <Badge
      // STATE_STYLE: domain state color palette — overrides shadcn Badge defaults
      // minWidth: uniform width across all states to prevent layout shift on state change
      // justify-center: needed because minWidth makes badge wider than content
      className={cn(STATE_STYLE.get(state), className)}
      style={{ minWidth: `${MIN_WIDTH}ch` }}
    >
      {Icon ? <Icon className={cn(state === "running" && "animate-spin")} /> : undefined}
      {state}
    </Badge>
  );
}
