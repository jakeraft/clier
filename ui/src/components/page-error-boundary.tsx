import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import type { ReactNode } from "react";
import { useLocation } from "react-router";
import { logger } from "@/lib/logger";
import { ErrorAlert } from "@/components/error-alert";

function PageFallback({ error }: Readonly<FallbackProps>) {
  return (
    <div className="p-6">
      <ErrorAlert message={error instanceof Error ? error.message : "An unexpected error occurred."} />
    </div>
  );
}

export function PageErrorBoundary({ children }: Readonly<{ children: ReactNode }>) {
  const location = useLocation();
  return (
    <ErrorBoundary
      FallbackComponent={PageFallback}
      onError={(error, info) => logger.error("Page render error", { error, componentStack: info.componentStack })}
      resetKeys={[location.pathname]}
    >
      {children}
    </ErrorBoundary>
  );
}
