import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utilities";
import { typography, typographyIcon } from "@/lib/typography";
import { flex, gap } from "@/lib/layout";
import { Card, CardHeader, CardTitle, CardAction, CardContent } from "@/components/ui/card";
import { EmptyState } from "@/components/empty-state";

interface SectionProperties {
  icon: LucideIcon;
  title: React.ReactNode;
  actions?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
  /** When provided, renders EmptyState instead of children. */
  empty?: { title: string; description: string };
}

export function SectionCard({ icon: Icon, title, actions, children, className, empty }: Readonly<SectionProperties>) {
  return (
    <Card className={cn(gap[1], className)}>
      <CardHeader>
        <CardTitle className={cn(flex.row, gap[1], typography[3])}>
          <Icon className={typographyIcon[3]} />
          {title}
        </CardTitle>
        <CardAction>{actions}</CardAction>
      </CardHeader>
      <CardContent>{empty ? <EmptyState title={empty.title} description={empty.description} /> : children}</CardContent>
    </Card>
  );
}
