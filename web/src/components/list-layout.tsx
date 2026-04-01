import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface ListLayoutProperties {
  error?: string;
  children: React.ReactNode;
}

export function ListLayout({ error, children }: Readonly<ListLayoutProperties>) {
  return (
    <div className={cn(flex.colFill, gap[3])}>
      {error && (
        <Alert className="bg-destructive/10 text-destructive border-destructive/20">
          <AlertCircle className="size-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {children}
    </div>
  );
}
