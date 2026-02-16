import { useEffect } from "react";
import { useIntelStore } from "@/features/intel";
import { IntelPanel } from "@/features/intel";
import {
  ContextMenu,
  MapLayoutSelect,
  MapPageShell,
  JumpbridgesToggle,
  MapZoomControls,
  MapCanvas,
  RegionSelect,
  useMapStore,
} from "@/features/map";
import NavbarSearch from "@/components/NavbarSearch";
import { UI_DIALOG } from "@/app/store/uiStore";
import { useAppModal } from "@/components/dialogs/AppModals";
import { useSettingsStore } from "@/app/store/settingsStore";
import IntelServerStatus from "./IntelServerStatus";
import {
  Activity,
  ChevronDown,
  ChevronUp,
  ScrollText,
  Upload,
} from "lucide-react";
import AlarmMuteToggleButton from "@/components/AlarmMuteToggleButton";

export default function Intel() {
  const reports = useIntelStore((state) => state.reports);
  const uploaders = useIntelStore((state) => state.uploaders);
  const version = useIntelStore((state) => state.version);
  const { open: openHelpModal } = useAppModal(UI_DIALOG.Help);
  const { open: openShareLinkModal } = useAppModal(UI_DIALOG.ShareLink);
  const mapViewMode = useSettingsStore((s) => s.settings.map.viewMode);
  const intelPanelOpen = useSettingsStore((s) => s.settings.intel.panelOpen);
  const alarmEnabled = useSettingsStore((s) => s.settings.alarm.enabled);
  const alarmVolume = useSettingsStore((s) => s.settings.alarm.volume);
  const applySetting = useSettingsStore((s) => s.apply);
  const alarmMuted = !alarmEnabled || alarmVolume <= 0;

  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

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

  const leftControls = (
    <>
      <NavbarSearch />
      <RegionSelect multi />
      <MapLayoutSelect inlineLabel="Layout" />
      <JumpbridgesToggle />
      <MapZoomControls />
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={openShareLinkModal}
      >
        Share
      </button>
      <button
        className="btn btn-xs btn-info btn-outline"
        onClick={openHelpModal}
      >
        Help
      </button>
    </>
  );

  const rightControls = (
    <>
      <IntelServerStatus version={version} />
      <span
        className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content"
        title="Active intel uploaders currently connected"
        aria-label="Active intel uploaders currently connected"
      >
        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-base-100/80 text-base-content">
          <Upload className="h-3.5 w-3.5" />
        </span>
        <span>{uploaders}</span>
      </span>
      <span
        className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content"
        title="Total reports currently loaded in the feed"
        aria-label="Total reports currently loaded in the feed"
      >
        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-base-100/80 text-base-content">
          <ScrollText className="h-3.5 w-3.5" />
        </span>
        <span>{reports.length}</span>
      </span>
      <AlarmMuteToggleButton
        muted={alarmMuted}
        onToggle={() => applySetting("alarm", "enabled", !alarmEnabled)}
      />
      {mapViewMode === "full" && (
        <button
          className="btn btn-xs btn-ghost"
          onClick={() => applySetting("intel", "panelOpen", !intelPanelOpen)}
          aria-label="Toggle intel panel"
        >
          {intelPanelOpen ? (
            <ChevronUp className="h-5 w-5" />
          ) : (
            <ChevronDown className="h-5 w-5" />
          )}
        </button>
      )}
    </>
  );

  return (
    <MapPageShell
      pageBadge={{ icon: <Activity className="h-3 w-3" />, label: "Intel" }}
      leftControls={leftControls}
      rightControls={rightControls}
      panel={<IntelPanel />}
      panelOpen={mapViewMode !== "full" || intelPanelOpen}
      panelClassName="w-96"
      onAutoHidePanel={() => applySetting("intel", "panelOpen", true)}
    >
      <MapCanvas />
      <ContextMenu />
    </MapPageShell>
  );
}
