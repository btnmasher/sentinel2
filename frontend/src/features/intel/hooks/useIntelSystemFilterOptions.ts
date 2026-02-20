import { useMemo } from "react";
import { useMapStore, useRegionNames } from "@/features/map";

type CoalescedRegionChip = {
  regionId: number;
  name: string;
  count: number;
  systemIds: number[];
};

export type CoalescedSystemChips = {
  regionChips: CoalescedRegionChip[];
  systemChips: number[];
};

type UseIntelSystemFilterOptionsInput = {
  selectedSystemIds: number[];
  jumpRange: number;
  setSystemFilters: (systemIds: number[]) => void;
};

export function useIntelSystemFilterOptions({
  selectedSystemIds,
  jumpRange,
  setSystemFilters,
}: UseIntelSystemFilterOptionsInput) {
  const systems = useMapStore((s) => s.systems);
  const regions = useMapStore((s) => s.regions);
  const gates = useMapStore((s) => s.gates);
  const characters = useMapStore((s) => s.characters);
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const characterLocations = useMapStore((s) => s.characterLocations);
  const { getRegionName } = useRegionNames();

  const activeCharacterIds =
    visibleCharacterIds.length > 0
      ? visibleCharacterIds
      : characters.map((char) => char.id);

  const activeLocations = activeCharacterIds
    .map((id) => characterLocations[id])
    .filter((location): location is number => typeof location === "number");

  const adjacency = useMemo(() => {
    const map = new Map<number, number[]>();
    gates
      .filter((gate) => gate.type === "solarsystem")
      .forEach((gate) => {
        if (!map.has(gate.from)) {
          map.set(gate.from, []);
        }
        if (!map.has(gate.to)) {
          map.set(gate.to, []);
        }
        map.get(gate.from)?.push(gate.to);
        map.get(gate.to)?.push(gate.from);
      });
    return map;
  }, [gates]);

  const applyRegionFilter = () => {
    if (activeLocations.length === 0) return;
    const regionIds = new Set(
      activeLocations
        .map((systemId) => systems[systemId]?.region)
        .filter((regionId): regionId is number => typeof regionId === "number"),
    );
    if (regionIds.size === 0) return;
    const systemIds = Object.values(systems)
      .filter((system) => regionIds.has(system.region))
      .map((system) => system.system);
    setSystemFilters(systemIds);
  };

  const applyJumpFilter = () => {
    if (activeLocations.length === 0) return;
    const maxJumps = Math.max(1, Math.min(20, Number(jumpRange) || 1));
    const visited = new Set<number>();
    const queue: Array<{ id: number; depth: number }> = [];
    activeLocations.forEach((id) => {
      visited.add(id);
      queue.push({ id, depth: 0 });
    });

    while (queue.length > 0) {
      const current = queue.shift();
      if (!current) break;
      if (current.depth >= maxJumps) continue;
      const neighbors = adjacency.get(current.id) ?? [];
      neighbors.forEach((next) => {
        if (visited.has(next)) return;
        visited.add(next);
        queue.push({ id: next, depth: current.depth + 1 });
      });
    }

    setSystemFilters(Array.from(visited));
  };

  const coalescedSystemChips = useMemo<CoalescedSystemChips>(() => {
    const selectedSet = new Set(selectedSystemIds);
    const byRegion = new Map<number, number[]>();
    Object.values(systems).forEach((system) => {
      const list = byRegion.get(system.region) ?? [];
      list.push(system.system);
      byRegion.set(system.region, list);
    });

    const regionChips: CoalescedRegionChip[] = [];
    const coveredSystemIds = new Set<number>();

    byRegion.forEach((regionSystemIds, regionId) => {
      if (regionSystemIds.length === 0) return;
      const isFullRegion = regionSystemIds.every((id) => selectedSet.has(id));
      if (!isFullRegion) return;
      const regionName =
        regions[regionId]?.name ??
        getRegionName(regionId, `Region ${regionId}`);
      regionChips.push({
        regionId,
        name: regionName,
        count: regionSystemIds.length,
        systemIds: regionSystemIds,
      });
      regionSystemIds.forEach((id) => coveredSystemIds.add(id));
    });

    return {
      regionChips,
      systemChips: selectedSystemIds.filter((id) => !coveredSystemIds.has(id)),
    };
  }, [regions, selectedSystemIds, systems]);

  return {
    activeLocations,
    applyJumpFilter,
    applyRegionFilter,
    coalescedSystemChips,
  };
}
