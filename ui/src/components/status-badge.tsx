import { Play, Square } from "lucide-react";
import { Badge } from "@/components/ui/badge";

const STATUS_CONFIG = new Map<string, { variant: "default" | "secondary"; icon: typeof Play }>([
  ["running", { variant: "default", icon: Play }],
  ["stopped", { variant: "secondary", icon: Square }],
]);

interface StatusBadgeProperties {
  status: string;
}

export function StatusBadge({ status }: Readonly<StatusBadgeProperties>) {
  const config = STATUS_CONFIG.get(status);
  const Icon = config?.icon;
  return (
    <Badge variant={config?.variant ?? "outline"}>
      {Icon ? <Icon /> : undefined}
      {status}
    </Badge>
  );
}
