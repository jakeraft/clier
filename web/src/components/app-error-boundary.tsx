import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import type { ReactNode } from "react";
import { logger } from "@/lib/logger";
import { cn } from "@/lib/utilities";
import { flex } from "@/lib/layout";
import { typography } from "@/lib/typography";

function AppFallback({ resetErrorBoundary }: Readonly<FallbackProps>) {
  return (
    <div className={cn(flex.center, "min-h-screen")}>
      <div className="text-center">
        <h1 className={cn(typography[2])}>Something went wrong</h1>
        <p className={cn("mt-2", typography[6])}>An unexpected error occurred.</p>
        <button
          type="button"
          className={cn("bg-primary text-primary-foreground mt-4 rounded-md px-4 py-2", typography[5])}
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
