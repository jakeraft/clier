import { ArchiveX } from "lucide-react";
import { cn } from "@/lib/utilities";
import { typography } from "@/lib/typography";
import { flex, gap } from "@/lib/layout";

interface EmptyStateProperties {
  title: string;
  description?: string;
  className?: string;
}

export function EmptyState({ title, description, className }: Readonly<EmptyStateProperties>) {
  return (
    <div className={cn(flex.center, flex.col, "flex-1 text-center", gap[1], typography[6], className)}>
      <ArchiveX className="size-5 opacity-50" />
      <span>{title}</span>
      {description ? <span className={typography[7]}>{description}</span> : undefined}
    </div>
  );
}
