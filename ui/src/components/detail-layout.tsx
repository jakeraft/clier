import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { Spinner } from "@/components/ui/spinner";
import { ErrorAlert } from "@/components/error-alert";

interface DetailLayoutProperties {
  error?: string;
  loading?: boolean;
  children: React.ReactNode;
}

export function DetailLayout({ error, loading, children }: Readonly<DetailLayoutProperties>) {
  return (
    <div className={cn(flex.col, gap[4])}>
      {error && <ErrorAlert message={error} />}
      {loading && !error && <Spinner className="mx-auto mt-8 size-6" />}
      {!loading && children}
    </div>
  );
}
