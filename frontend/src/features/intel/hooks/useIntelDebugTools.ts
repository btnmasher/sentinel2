import { useEffect } from "react";
import { useIntelStore } from "@/features/intel";
import { useMapStore } from "@/features/map";

type FakeIntelOptions = {
  count?: number;
  maxAgeMinutes?: number;
  includeAdjacent?: boolean;
  clearExisting?: boolean;
  resetFilters?: boolean;
  clearRatio?: number;
};

type DebugAPI = {
  generateFakeIntelReports?: (options?: FakeIntelOptions) => {
    created: number;
    cleared: number;
    regions: number[];
    adjacentRegion?: number;
  };
};

export default function useIntelDebugTools() {
  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

    const win = window as Window & { sentinelDebug?: DebugAPI };
    const api = win.sentinelDebug ?? {};

    const pickRandom = <T,>(items: T[]): T =>
      items[Math.floor(Math.random() * items.length)];

    const generateFakeIntelReports = (options?: FakeIntelOptions) => {
      const count = Math.max(1, Math.min(500, options?.count ?? 40));
      const maxAgeMinutes = Math.max(
        0,
        Math.min(120, options?.maxAgeMinutes ?? 14),
      );
      const includeAdjacent = options?.includeAdjacent ?? false;
      const resetFilters = options?.resetFilters ?? true;
      const clearRatio = Math.max(0, Math.min(0.6, options?.clearRatio ?? 0.2));

      const mapState = useMapStore.getState();
      const intelState = useIntelStore.getState();

      const loadedRegions = new Set(
        mapState.mapRegions
          .map((value) => Number.parseInt(value, 10))
          .filter((value) => Number.isFinite(value)),
      );
      if (loadedRegions.size === 0) {
        const result = { created: 0, cleared: 0, regions: [] };
        console.warn(
          "[sentinelDebug] no loaded regions; load at least one region first",
          result,
        );
        return result;
      }

      const adjacent = new Set<number>();
      for (const gate of mapState.gates) {
        const fromLoaded = loadedRegions.has(gate.from_region);
        const toLoaded = loadedRegions.has(gate.to_region);
        if (fromLoaded && !toLoaded) {
          adjacent.add(gate.to_region);
        }
        if (toLoaded && !fromLoaded) {
          adjacent.add(gate.from_region);
        }
      }

      let chosenAdjacent: number | undefined;
      if (includeAdjacent && adjacent.size > 0) {
        chosenAdjacent = pickRandom(Array.from(adjacent));
      }

      const regionScope = new Set(loadedRegions);
      if (chosenAdjacent) {
        regionScope.add(chosenAdjacent);
      }

      const scopedSystems = Object.values(mapState.systems).filter((system) =>
        regionScope.has(system.region),
      );
      if (scopedSystems.length === 0) {
        const result = {
          created: 0,
          cleared: 0,
          regions: Array.from(regionScope),
          adjacentRegion: chosenAdjacent,
        };
        console.warn(
          "[sentinelDebug] no loaded systems for selected region scope",
          result,
        );
        return result;
      }

      const authors = [
        "Debug Scout",
        "Fake FC",
        "Synthetic Pilot",
        "Layout Tester",
        "QA Alt",
      ];
      const nowSeconds = Math.floor(Date.now() / 1000);
      const batchID = Date.now();
      const fakeReports: Array<{
        id: number;
        recordId: string;
        time: number;
        author: string;
        text: string;
        systems: Array<{
          system: number;
          name: string;
          constellation: number;
          region: number;
        }>;
        regions: number[];
        channel_id: string;
      }> = [];
      const threatCandidates: Array<{
        id: number;
        time: number;
        systems: Array<{
          system: number;
          name: string;
          constellation: number;
          region: number;
        }>;
      }> = [];
      let cleared = 0;

      for (let i = 0; i < count; i++) {
        const shouldGenerateClear =
          threatCandidates.length > 0 && Math.random() < clearRatio;
        if (shouldGenerateClear) {
          const targetReport = pickRandom(threatCandidates);
          const targetSystem = pickRandom(targetReport.systems);
          const clearTime = Math.min(
            nowSeconds,
            targetReport.time + 30 + Math.floor(Math.random() * 240),
          );

          fakeReports.push({
            id: batchID + i,
            recordId: `debug-${batchID}-${i}-${Math.random().toString(36).slice(2, 8)}`,
            time: clearTime,
            author: pickRandom(authors),
            text: `${targetSystem.name} CLR`,
            systems: [targetSystem],
            regions: [targetSystem.region],
            channel_id: "debug",
          });
          cleared++;
          continue;
        }

        const picks = new Map<number, (typeof scopedSystems)[number]>();
        while (picks.size < 1 && picks.size < scopedSystems.length) {
          const candidate = pickRandom(scopedSystems);
          picks.set(candidate.system, candidate);
        }
        const pickedSystems = Array.from(picks.values());
        const pickedRegions = Array.from(
          new Set(pickedSystems.map((s) => s.region)),
        );
        const ageSeconds = Math.floor(Math.random() * (maxAgeMinutes * 60 + 1));
        const reportTime = nowSeconds - ageSeconds;
        const lead = pickedSystems[0];
        const suffix =
          pickedSystems.length > 1 ? ` +${pickedSystems.length - 1}` : "";

        const threatReport = {
          id: batchID + i,
          recordId: `debug-${batchID}-${i}-${Math.random().toString(36).slice(2, 8)}`,
          time: reportTime,
          author: pickRandom(authors),
          text: `${lead.name} TEST contact${suffix}`,
          systems: pickedSystems.map((system) => ({
            system: system.system,
            name: system.name,
            constellation: system.constellation,
            region: system.region,
          })),
          regions: pickedRegions,
          channel_id: "debug",
        };
        fakeReports.push(threatReport);
        threatCandidates.push({
          id: threatReport.id,
          time: threatReport.time,
          systems: threatReport.systems,
        });
      }

      const existingReports = options?.clearExisting ? [] : intelState.reports;
      intelState.setReports([...fakeReports, ...existingReports]);
      if (resetFilters) {
        intelState.setLogFilters({
          includeSystemLogs: true,
          includeSystemAlarm: true,
          includeUnknownLogs: true,
          includeUnloadedRegionsLogs: true,
          system: [],
        });
      }

      const result = {
        created: fakeReports.length,
        cleared,
        regions: Array.from(regionScope),
        adjacentRegion: chosenAdjacent,
      };
      console.info("[sentinelDebug] generated fake intel reports", result);
      return result;
    };

    api.generateFakeIntelReports = generateFakeIntelReports;
    win.sentinelDebug = api;

    return () => {
      const current = win.sentinelDebug;
      if (current?.generateFakeIntelReports === generateFakeIntelReports) {
        delete current.generateFakeIntelReports;
      }
    };
  }, []);
}
