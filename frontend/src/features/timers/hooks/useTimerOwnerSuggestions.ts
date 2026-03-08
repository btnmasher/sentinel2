import { useCallback } from "react";
import { api } from "@/config/api";
import type { TimerEntityOption } from "../types";

export function useTimerOwnerSuggestions() {
  return useCallback(async (query: string, signal?: AbortSignal) => {
    const response = await api.get<{ entities: TimerEntityOption[] }>(
      `/organizations/search?query=${encodeURIComponent(query)}`,
      { signal },
    );
    return response.data.entities || [];
  }, []);
}
