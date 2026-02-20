import { useMemo } from "react";
import { REGION_MAP, REGIONS } from "../types/regions";

type UseRegionNamesResult = {
  regionNameById: Map<string, string>;
  getRegionName: (regionId: number | string, fallback?: string) => string;
};

export function useRegionNames(): UseRegionNamesResult {
  const regionNameById = useMemo(
    () =>
      new Map(
        REGIONS.map((region) => [String(region.region), region.name] as const),
      ),
    [],
  );

  const getRegionName = (regionId: number | string, fallback = "") => {
    const key = String(regionId);
    return regionNameById.get(key) || REGION_MAP[key] || fallback;
  };

  return {
    regionNameById,
    getRegionName,
  };
}
