import { useEffect, useMemo, useRef, useState } from "react";
import { ESI_BASE } from "@/config/esi";
import { pb } from "@/config/pb";
import { useIntelStore } from "../store/intelStore";
import { useMapStore } from "@/features/map";
import IntelFiltersCard from "./IntelFiltersCard";
import IntelCharacterVisibilityCard from "./IntelCharacterVisibilityCard";
import IntelFeedCard from "./IntelFeedCard";
import { useSettingsStore } from "@/app/store/settingsStore";

export default function IntelPanel() {
  const reports = useIntelStore((s) => s.reports);
  const logFilters = useIntelStore((s) => s.logFilters);
  const toggleSystemFilter = useIntelStore((s) => s.toggleSystemFilter);
  const setLogFilters = useIntelStore((s) => s.setLogFilters);

  const systems = useMapStore((s) => s.systems);
  const regions = useMapStore((s) => s.regions);
  const [systemNames, setSystemNames] = useState<Record<number, string>>({});
  const [channelNames, setChannelNames] = useState<Record<string, string>>({});
  const systemNamesRef = useRef(systemNames);
  const { panelOpen, filtersOpen, charactersOpen, feedOpen } = useSettingsStore(
    (s) => s.settings.intel,
  );
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const applySetting = useSettingsStore((s) => s.apply);

  useEffect(() => {
    systemNamesRef.current = systemNames;
  }, [systemNames]);

  useEffect(() => {
    if (logFilters.system.length === 0) return;
    const currentNames = systemNamesRef.current;
    const missing = logFilters.system.filter((id) => !currentNames[id]);
    if (!missing.length) return;

    const unloaded: number[] = [];
    let hasUpdates = false;
    setSystemNames((prev) => {
      const next = { ...prev };
      missing.forEach((id) => {
        const system = systems[id];
        if (system) {
          if (next[id] !== system.name) {
            next[id] = system.name;
            hasUpdates = true;
          }
        } else {
          unloaded.push(id);
        }
      });
      return hasUpdates ? next : prev;
    });

    if (unloaded.length) {
      fetch(`${ESI_BASE}/v2/universe/names/`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(unloaded),
      })
        .then((res) => res.json())
        .then((data) => {
          setSystemNames((prev) => {
            const updated = { ...prev };
            let changed = false;
            data.forEach((entry: { id: number; name: string }) => {
              if (updated[entry.id] !== entry.name) {
                updated[entry.id] = entry.name;
                changed = true;
              }
            });
            return changed ? updated : prev;
          });
        });
    }
  }, [logFilters.system, systems]);

  useEffect(() => {
    let active = true;
    pb.collection("intel_channels")
      .getFullList({ sort: "channel_name" })
      .then((records) => {
        if (!active) return;
        const next: Record<string, string> = {};
        records.forEach((record) => {
          if (!record.id) return;
          const name = String(record.channel_name || "").trim();
          if (name) {
            next[record.id] = name;
          }
        });
        setChannelNames(next);
      })
      .catch(() => {
        if (active) {
          setChannelNames({});
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const filteredLogs = useMemo(() => {
    const regionIds = new Set(
      Object.keys(regions).map((id) => parseInt(id, 10)),
    );
    const systemSet = new Set(logFilters.system);

    return reports.filter((log) => {
      const unknownLocation = log.systems.length === 0;
      const loadedRegion = log.regions.some((region) => regionIds.has(region));
      const hasSystemFiltered = log.systems.some((system) =>
        systemSet.has(system.system),
      );

      if (logFilters.system.length) {
        return hasSystemFiltered;
      }
      if (!logFilters.includeUnknownLogs && unknownLocation) {
        return false;
      }
      if (
        !logFilters.includeUnloadedRegionsLogs &&
        !loadedRegion &&
        !unknownLocation
      ) {
        return false;
      }
      return true;
    });
  }, [logFilters, regions, reports]);

  const forceVisible = mapViewMode !== "full";
  if (!forceVisible && !panelOpen) {
    return null;
  }

  return (
    <div className="flex h-full min-h-0 max-h-full flex-col">
      <div className="shrink-0">
        <h2 className="text-lg font-display leading-none">Intel Console</h2>
        <p className="mt-1 text-xs text-slate-400">
          Filters and live report feed
        </p>
      </div>
      <div className="mt-4 min-h-0 flex flex-1 flex-col gap-3">
        <div className="shrink-0">
          <IntelFiltersCard
            logFilters={logFilters}
            setLogFilters={setLogFilters}
            toggleSystemFilter={toggleSystemFilter}
            systemNames={systemNames}
            open={filtersOpen}
            onToggle={() => applySetting("intel", "filtersOpen", !filtersOpen)}
          />
        </div>
        <div className="shrink-0">
          <IntelCharacterVisibilityCard
            open={charactersOpen}
            onToggle={() =>
              applySetting("intel", "charactersOpen", !charactersOpen)
            }
          />
        </div>
        <div className="min-h-0 flex flex-1 flex-col overflow-hidden">
          <IntelFeedCard
            logs={filteredLogs}
            channelNames={channelNames}
            open={feedOpen}
            onToggle={() => applySetting("intel", "feedOpen", !feedOpen)}
          />
        </div>
      </div>
    </div>
  );
}
