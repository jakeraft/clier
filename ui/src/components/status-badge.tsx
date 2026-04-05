import { Play, Square } from "lucide-react";
import { Badge } from "@/components/ui/badge";

const STATUS_CONFIG = new Map<string, { className: string; icon: typeof Play }>([
  ["running", { className: "bg-success text-success-foreground", icon: Play }],
  ["stopped", { className: "", icon: Square }],
]);

interface StatusBadgeProperties {
  status: string;
}

export function StatusBadge({ status }: Readonly<StatusBadgeProperties>) {
  const config = STATUS_CONFIG.get(status);
  const Icon = config?.icon;
  return (
    <Badge variant={config ? "secondary" : "outline"} className={config?.className}>
      {Icon ? <Icon /> : undefined}
      {status}
    </Badge>
  );
}
