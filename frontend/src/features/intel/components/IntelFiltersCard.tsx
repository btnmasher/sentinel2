type IntelFilters = {
  includeUnknownLogs: boolean;
  includeUnknownAlarm: boolean;
  includeUnloadedRegionsLogs: boolean;
  includeUnloadedRegionsAlarm: boolean;
  system: number[];
};

type IntelFiltersCardProps = {
  logFilters: IntelFilters;
  setLogFilters: (filters: Partial<IntelFilters>) => void;
  toggleSystemFilter: (systemId: number) => void;
  systemNames: Record<number, string>;
  open: boolean;
  onToggle: () => void;
};

const filterConfig: Array<{
  key: "includeUnknown" | "includeUnloadedRegions";
  label: string;
}> = [
  {
    key: "includeUnknown",
    label: "Show intel from unknown locations",
  },
  {
    key: "includeUnloadedRegions",
    label: "Show intel from regions not visible",
  },
];

export default function IntelFiltersCard({
  logFilters,
  setLogFilters,
  toggleSystemFilter,
  systemNames,
  open,
  onToggle,
}: IntelFiltersCardProps) {
  const systems = useMapStore((s) => s.systems);
  const gates = useMapStore((s) => s.gates);
  const characters = useMapStore((s) => s.characters);
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const characterLocations = useMapStore((s) => s.characterLocations);
  const [jumpRange, setJumpRange] = useState(3);

  const toggleFilter = (
    key: "includeUnknown" | "includeUnloadedRegions",
    target: "Logs" | "Alarm",
  ) => {
    const filterKey = `${key}${target}` as keyof IntelFilters;
    setLogFilters({ [filterKey]: !logFilters[filterKey] });
  };

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
    setLogFilters({ system: systemIds });
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

    setLogFilters({ system: Array.from(visited) });
  };

  return (
    <AccordionCard
      title="Intel Filters"
      subtitle="Shift-click systems to filter"
      open={open}
      onToggle={onToggle}
    >
      <div className="space-y-4">
        <div className="space-y-2">
          {filterConfig.map((filter) => (
            <div
              key={filter.key}
              className="flex items-center justify-between text-xs"
            >
              <span>{filter.label}</span>
              <div className="flex items-center gap-2">
                {(["Logs", "Alarm"] as const).map((target) => {
                  const filterKey =
                    `${filter.key}${target}` as keyof IntelFilters;
                  return (
                    <button
                      key={target}
                      className={`btn btn-xs ${
                        logFilters[filterKey] ? "btn-info" : "btn-ghost"
                      }`}
                      onClick={() => toggleFilter(filter.key, target)}
                      disabled={logFilters.system.length > 0}
                    >
                      {target}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
        <div className="divider">System Filters</div>
        <div className="space-y-2 text-xs">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-slate-400">From character locations:</span>
            <button
              className="btn btn-xs btn-outline"
              onClick={applyRegionFilter}
              disabled={activeLocations.length === 0}
            >
              Same region
            </button>
            <div className="flex items-center gap-2">
              <button
                className="btn btn-xs btn-outline"
                onClick={applyJumpFilter}
                disabled={activeLocations.length === 0}
              >
                Within
              </button>
              <input
                type="number"
                min={1}
                max={20}
                value={jumpRange}
                onChange={(event) =>
                  setJumpRange(
                    Math.max(1, Math.min(20, Number(event.target.value) || 1)),
                  )
                }
                className="input input-xs w-16"
              />
              <span className="text-slate-400">jumps</span>
            </div>
          </div>
          {activeLocations.length === 0 && (
            <div className="text-[11px] text-slate-500">
              No character locations available.
            </div>
          )}
        </div>
        <div className="flex flex-wrap gap-2 text-xs">
          {logFilters.system.length > 0 ? (
            logFilters.system.map((id) => (
              <button
                key={id}
                className="badge badge-success badge-sm"
                onClick={() => toggleSystemFilter(id)}
              >
                {systemNames[id] || id}
              </button>
            ))
          ) : (
            <span className="text-slate-500">No systems selected</span>
          )}
        </div>
      </div>
    </AccordionCard>
  );
}
import AccordionCard from "@/components/AccordionCard";
import { useMapStore } from "@/features/map";
import { useMemo, useState } from "react";
