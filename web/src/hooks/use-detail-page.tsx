import { useEffect, useState } from "react";
import { logger } from "@/lib/logger";
import { getErrorMessage } from "@/lib/utilities";

export function useDetailPage<T>(entityId: string | undefined, fetch: (id: string) => Promise<T>) {
  const [data, setData] = useState<T | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!entityId) return;
    let current = true;
    setData(undefined);
    setError(undefined);
    setLoading(true);
    fetch(entityId)
      .then((result) => {
        if (current) setData(result);
      })
      .catch((error_: unknown) => {
        if (current) {
          logger.error("Failed to load detail", { error: error_ });
          setError(getErrorMessage(error_, "Failed to load"));
        }
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [entityId, fetch]);

  return { data, error, loading };
}
