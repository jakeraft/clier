import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import type { ReactNode } from "react";
import { logger } from "@/lib/logger";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { typography } from "@/lib/typography";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

function AppFallback({ error, resetErrorBoundary }: Readonly<FallbackProps>) {
  return (
    <div className="p-6">
      <Alert variant="destructive">
        <AlertTitle>Something went wrong</AlertTitle>
        <AlertDescription>{error instanceof Error ? error.message : "An unexpected error occurred."}</AlertDescription>
      </Alert>
      <div className={cn("mt-4", flex.row, gap[3])}>
        <button
          type="button"
          className={cn("bg-primary text-primary-foreground rounded-md px-4 py-2", typography[5])}
          onClick={resetErrorBoundary}
        >
          Refresh
        </button>
      </div>
    </div>
  );
}

export function AppErrorBoundary({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <ErrorBoundary
      FallbackComponent={AppFallback}
      onError={(error, info) => logger.error("Unhandled render error", { error, componentStack: info.componentStack })}
      onReset={() => globalThis.location.reload()}
    >
      {children}
    </ErrorBoundary>
  );
}
