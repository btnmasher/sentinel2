import { useState } from "react";
import AccordionCard from "@/components/AccordionCard";
import { useIntelSystemFilterOptions } from "../hooks/useIntelSystemFilterOptions";

type IntelFilters = {
  includeSystemLogs: boolean;
  includeSystemAlarm: boolean;
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

const FILTER_CONFIG: Array<{
  key: "includeUnknown" | "includeUnloadedRegions";
  label: string;
}> = [
  { key: "includeUnknown", label: "Show intel from unknown locations" },
  {
    key: "includeUnloadedRegions",
    label: "Show intel from regions not visible",
  },
];

function FilterToggleRows({
  logFilters,
  onToggleFilter,
}: {
  logFilters: IntelFilters;
  onToggleFilter: (
    key: "includeUnknown" | "includeUnloadedRegions",
    target: "Logs" | "Alarm",
  ) => void;
}) {
  return (
    <div className="space-y-2">
      {FILTER_CONFIG.map((filter) => (
        <div
          key={filter.key}
          className="flex items-center justify-between text-xs"
        >
          <span>{filter.label}</span>
          <div className="flex items-center gap-2">
            {(["Logs", "Alarm"] as const).map((target) => {
              const filterKey = `${filter.key}${target}` as keyof IntelFilters;
              return (
                <button
                  key={target}
                  className={`btn btn-xs ${
                    logFilters[filterKey] ? "btn-info" : "btn-ghost"
                  }`}
                  onClick={() => onToggleFilter(filter.key, target)}
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
  );
}

function SystemSelectionChips({
  logFilters,
  systemNames,
  toggleSystemFilter,
  setLogFilters,
  coalescedSystemChips,
}: {
  logFilters: IntelFilters;
  systemNames: Record<number, string>;
  toggleSystemFilter: (systemId: number) => void;
  setLogFilters: (filters: Partial<IntelFilters>) => void;
  coalescedSystemChips: ReturnType<
    typeof useIntelSystemFilterOptions
  >["coalescedSystemChips"];
}) {
  if (logFilters.system.length === 0) {
    return <span className="text-slate-500">No systems selected</span>;
  }

  return (
    <>
      {coalescedSystemChips.regionChips.map((region) => (
        <button
          key={`region-${region.regionId}`}
          className="badge badge-info badge-sm"
          onClick={() =>
            setLogFilters({
              system: logFilters.system.filter(
                (id) => !region.systemIds.includes(id),
              ),
            })
          }
        >
          {region.name} ({region.count})
        </button>
      ))}
      {coalescedSystemChips.systemChips.map((id) => (
        <button
          key={id}
          className="badge badge-success badge-sm"
          onClick={() => toggleSystemFilter(id)}
        >
          {systemNames[id] || id}
        </button>
      ))}
    </>
  );
}

export default function IntelFiltersCard({
  logFilters,
  setLogFilters,
  toggleSystemFilter,
  systemNames,
  open,
  onToggle,
}: IntelFiltersCardProps) {
  const [jumpRange, setJumpRange] = useState(3);
  const {
    activeLocations,
    applyJumpFilter,
    applyRegionFilter,
    coalescedSystemChips,
  } = useIntelSystemFilterOptions({
    selectedSystemIds: logFilters.system,
    jumpRange,
    setSystemFilters: (systemIds) => setLogFilters({ system: systemIds }),
  });

  const toggleFilter = (
    key: "includeUnknown" | "includeUnloadedRegions",
    target: "Logs" | "Alarm",
  ) => {
    const filterKey = `${key}${target}` as keyof IntelFilters;
    setLogFilters({ [filterKey]: !logFilters[filterKey] });
  };

  return (
    <AccordionCard
      title="Intel Filters"
      subtitle="Shift-click systems to filter"
      open={open}
      onToggle={onToggle}
    >
      <div className="space-y-4">
        <FilterToggleRows
          logFilters={logFilters}
          onToggleFilter={toggleFilter}
        />
        <div className="divider">System Filters</div>
        <div className="space-y-2 text-xs">
          <div className="flex items-center justify-between">
            <span className="text-slate-400">Apply system filters to:</span>
            <div className="flex items-center gap-2">
              {(["Logs", "Alarm"] as const).map((target) => {
                const filterKey =
                  `includeSystem${target}` as keyof IntelFilters;
                return (
                  <button
                    key={target}
                    className={`btn btn-xs ${
                      logFilters[filterKey] ? "btn-info" : "btn-ghost"
                    }`}
                    onClick={() =>
                      setLogFilters({ [filterKey]: !logFilters[filterKey] })
                    }
                  >
                    {target}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-slate-400">From character locations:</span>
            <button
              className="btn btn-xs btn-ghost"
              onClick={() => setLogFilters({ system: [] })}
              disabled={logFilters.system.length === 0}
            >
              Clear systems
            </button>
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
          <SystemSelectionChips
            logFilters={logFilters}
            systemNames={systemNames}
            toggleSystemFilter={toggleSystemFilter}
            setLogFilters={setLogFilters}
            coalescedSystemChips={coalescedSystemChips}
          />
        </div>
      </div>
    </AccordionCard>
  );
}
