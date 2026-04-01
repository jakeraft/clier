import { ErrorBoundary, type FallbackProps } from "react-error-boundary";
import type { ReactNode } from "react";
import { useLocation, useNavigate } from "react-router";
import { logger } from "@/lib/logger";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { typography } from "@/lib/typography";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

function PageFallback({ error, resetErrorBoundary }: Readonly<FallbackProps>) {
  const navigate = useNavigate();
  return (
    <div className="p-6">
      <Alert variant="destructive">
        <AlertTitle>Failed to load this page</AlertTitle>
        <AlertDescription>{error instanceof Error ? error.message : "An unexpected error occurred."}</AlertDescription>
      </Alert>
      <div className={cn("mt-4", flex.row, gap[3])}>
        <button
          type="button"
          className={cn("bg-primary text-primary-foreground rounded-md px-4 py-2", typography[5])}
          onClick={resetErrorBoundary}
        >
          Retry
        </button>
        <button
          type="button"
          className={cn("rounded-md border px-4 py-2", typography[5])}
          onClick={() => {
            void navigate("/");
          }}
        >
          Go to home
        </button>
      </div>
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
