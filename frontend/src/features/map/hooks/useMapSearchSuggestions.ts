import { useCallback } from "react";
import { api } from "@/config/api";
import { REGIONS } from "../types/regions";
import { useRegionNames } from "./useRegionNames";

type SystemResult = {
  id: number;
  name: string;
  region: string;
  region_id: number;
  kind: "system";
};

type RegionResult = {
  id: string;
  name: string;
  kind: "region";
};

export type MapSearchSuggestion = SystemResult | RegionResult;

type UseMapSearchSuggestionsOptions = {
  includeRegions?: boolean;
};

export function useMapSearchSuggestions({
  includeRegions = false,
}: UseMapSearchSuggestionsOptions = {}) {
  const { getRegionName } = useRegionNames();

  const loadSuggestions = useCallback(
    async (
      query: string,
      signal?: AbortSignal,
    ): Promise<MapSearchSuggestion[]> => {
      const trimmed = query.trim();
      const lower = trimmed.toLowerCase();

      const regionMatches: RegionResult[] = includeRegions
        ? REGIONS.filter((region) => region.name.toLowerCase().includes(lower))
            .slice(0, 6)
            .map((region) => ({
              id: String(region.region),
              name: region.name,
              kind: "region",
            }))
        : [];

      const response = await api.get<{
        systems: Array<{
          id: number;
          name: string;
          region: string;
          region_id: number;
        }>;
      }>(`/map/search?q=${encodeURIComponent(trimmed)}`, { signal });

      const systemMatches: SystemResult[] = (response.data.systems || []).map(
        (system) => ({
          ...system,
          region:
            system.region || getRegionName(system.region_id, "Unknown Region"),
          kind: "system",
        }),
      );

      return [...regionMatches, ...systemMatches];
    },
    [getRegionName, includeRegions],
  );

  return { loadSuggestions };
}
