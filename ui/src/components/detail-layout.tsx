import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { Spinner } from "@/components/ui/spinner";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface DetailLayoutProperties {
  error?: string;
  loading?: boolean;
  children: React.ReactNode;
}

export function DetailLayout({ error, loading, children }: Readonly<DetailLayoutProperties>) {
  return (
    <div className={cn(flex.col, gap[4])}>
      {error && (
        <Alert className="bg-destructive/10 text-destructive border-destructive/20">
          <AlertCircle className="size-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {loading && !error && <Spinner className="mx-auto mt-8 size-6" />}
      {!loading && children}
    </div>
  );
}
