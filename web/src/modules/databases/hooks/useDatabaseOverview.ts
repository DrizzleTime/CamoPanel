import { useEffect, useState } from "react";
import { getDatabaseOverview } from "../api";
import type { DatabaseOverview } from "../types";

export function useDatabaseOverview(selectedInstanceId?: string) {
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [loadingOverview, setLoadingOverview] = useState(false);

  const refresh = async (instanceId = selectedInstanceId) => {
    if (!instanceId) {
      setOverview(null);
      return null;
    }

    setLoadingOverview(true);
    try {
      const response = await getDatabaseOverview(instanceId);
      setOverview(response);
      return response;
    } finally {
      setLoadingOverview(false);
    }
  };

  useEffect(() => {
    if (!selectedInstanceId) {
      setOverview(null);
      return;
    }
    void refresh(selectedInstanceId);
  }, [selectedInstanceId]);

  return {
    overview,
    loadingOverview,
    refresh,
  };
}
