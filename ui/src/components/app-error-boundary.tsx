import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import type { ReactNode } from "react";
import { logger } from "@/lib/logger";
import { ErrorAlert } from "@/components/error-alert";

function AppFallback({ error }: Readonly<FallbackProps>) {
  return (
    <div className="p-6">
      <ErrorAlert message={error instanceof Error ? error.message : "An unexpected error occurred."} />
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
