import { useEffect, useState } from "react";
import { logger } from "@/lib/logger";
import { getErrorMessage } from "@/lib/utilities";

export function useEntityList<T>(fetchList: () => Promise<T[]>) {
  const [items, setItems] = useState<T[]>([]);
  const [error, setError] = useState<string | undefined>();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let current = true;
    fetchList()
      .then((result) => {
        if (current) setItems(result);
      })
      .catch((error_: unknown) => {
        if (current) {
          logger.error("Failed to load list", { error: error_ });
          setError(getErrorMessage(error_, "Failed to load"));
        }
      })
      .finally(() => {
        if (current) setReady(true);
      });
    return () => {
      current = false;
    };
  }, [fetchList]);

  return { items, ready, error };
}
