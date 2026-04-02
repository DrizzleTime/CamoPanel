import { useEffect, useState } from "react";
import { listDatabaseInstances } from "../api";
import type { DatabaseEngine, DatabaseInstance } from "../types";

export function useDatabaseInstances(engine: DatabaseEngine) {
  const [instances, setInstances] = useState<DatabaseInstance[]>([]);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>();
  const [loadingInstances, setLoadingInstances] = useState(true);

  const refresh = async (preferredId?: string) => {
    setLoadingInstances(true);
    try {
      const response = await listDatabaseInstances(engine);
      setInstances(response.items);
      const nextSelectedId =
        (preferredId && response.items.some((item) => item.id === preferredId) ? preferredId : undefined) ??
        (response.items[0]?.id || undefined);
      setSelectedInstanceId(nextSelectedId);
      return nextSelectedId;
    } finally {
      setLoadingInstances(false);
    }
  };

  useEffect(() => {
    void refresh();
  }, [engine]);

  return {
    instances,
    selectedInstanceId,
    setSelectedInstanceId,
    loadingInstances,
    refresh,
  };
}
