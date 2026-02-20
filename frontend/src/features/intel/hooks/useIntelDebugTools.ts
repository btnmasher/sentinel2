import { useEffect } from "react";
import { useIntelStore } from "@/features/intel";
import { useMapStore } from "@/features/map";

type DebugSystem = {
  system: number;
  name: string;
  constellation: number;
  region: number;
};

type FakeIntelOptions = {
  count?: number;
  maxAgeMinutes?: number;
  includeAdjacent?: boolean;
  clearExisting?: boolean;
  resetFilters?: boolean;
  clearRatio?: number;
};

type FakeTimerOptions = {
  count?: number;
  maxHoursAhead?: number;
  imminentRatio?: number;
  includeAdjacent?: boolean;
  clearExisting?: boolean;
  includeAnsiblex?: boolean;
  disableJumpbridges?: boolean;
};

type FakeScenarioOptions = {
  intel?: FakeIntelOptions;
  timers?: FakeTimerOptions;
};

type FakeIntelResult = {
  created: number;
  cleared: number;
  regions: number[];
  adjacentRegion?: number;
};

type FakeTimersResult = {
  created: number;
  regions: number[];
  adjacentRegion?: number;
  ansiblexSystems: number[];
  disabledJumpbridges: number;
};

type FakeTimerPreview = {
  title: string;
  next_expires_at: string;
  severity: string;
  standing_type: string;
  timer_kind: string;
  structure_type: string;
  stage_label: string;
  planet_name?: string;
  moon_name?: string;
  skyhook_fullness_pct?: number;
};

type FakeTimerSignal = {
  system_id: number;
  count: number;
  remaining_count: number;
  next_expires_at: string;
  severity: string;
  standing_type: string;
  timer_kind: string;
  structure_type: string;
  stage_label: string;
  title: string;
  skyhook_fullness_pct?: number;
  timers: FakeTimerPreview[];
};

type DebugAPI = {
  generateFakeIntelReports?: (options?: FakeIntelOptions) => FakeIntelResult;
  generateFakeTimerSignals?: (options?: FakeTimerOptions) => FakeTimersResult;
  generateFakeIntelAndTimerScenario?: (options?: FakeScenarioOptions) => {
    intel: FakeIntelResult;
    timers: FakeTimersResult;
  };
  [key: string]: unknown;
};

export default function useIntelDebugTools() {
  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

    const win = window as Window & { sentinelDebug?: DebugAPI };
    const api = win.sentinelDebug ?? {};

    const pickRandom = <T>(items: readonly T[]): T =>
      items[Math.floor(Math.random() * items.length)];

    const resolveScopedSystems = (includeAdjacent: boolean) => {
      const mapState = useMapStore.getState();
      const loadedRegions = new Set(
        mapState.mapRegions
          .map((value) => Number.parseInt(value, 10))
          .filter((value) => Number.isFinite(value)),
      );
      if (loadedRegions.size === 0) {
        return {
          ok: false as const,
          result: { regions: [] as number[], adjacentRegion: undefined },
        };
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

      const scopedSystems: DebugSystem[] = Object.values(
        mapState.systems,
      ).filter((system) => regionScope.has(system.region));
      const scopeInfo = {
        regions: Array.from(regionScope),
        adjacentRegion: chosenAdjacent,
      };
      if (scopedSystems.length === 0) {
        return {
          ok: false as const,
          result: scopeInfo,
        };
      }
      return {
        ok: true as const,
        mapState,
        scopedSystems,
        scopeInfo,
      };
    };

    const generateFakeIntelReports = (
      options?: FakeIntelOptions,
    ): FakeIntelResult => {
      const count = Math.max(1, Math.min(500, options?.count ?? 40));
      const maxAgeMinutes = Math.max(
        0,
        Math.min(120, options?.maxAgeMinutes ?? 14),
      );
      const includeAdjacent = options?.includeAdjacent ?? false;
      const resetFilters = options?.resetFilters ?? true;
      const clearRatio = Math.max(0, Math.min(0.6, options?.clearRatio ?? 0.2));

      const scoped = resolveScopedSystems(includeAdjacent);
      if (!scoped.ok) {
        const result: FakeIntelResult = {
          created: 0,
          cleared: 0,
          regions: scoped.result.regions,
          adjacentRegion: scoped.result.adjacentRegion,
        };
        if (result.regions.length === 0) {
          console.warn(
            "[sentinelDebug] no loaded regions; load at least one region first",
            result,
          );
        } else {
          console.warn(
            "[sentinelDebug] no loaded systems for selected region scope",
            result,
          );
        }
        return result;
      }

      const { scopedSystems, scopeInfo } = scoped;
      const intelState = useIntelStore.getState();

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
            text: `${targetSystem.name} clear, gate green and local quiet`,
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
          text: `${lead.name} TEST contact in system, possible hostile fleet movement${suffix}`,
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

      const result: FakeIntelResult = {
        created: fakeReports.length,
        cleared,
        regions: scopeInfo.regions,
        adjacentRegion: scopeInfo.adjacentRegion,
      };
      console.info("[sentinelDebug] generated fake intel reports", result);
      return result;
    };

    const generateFakeTimerSignals = (
      options?: FakeTimerOptions,
    ): FakeTimersResult => {
      const count = Math.max(1, Math.min(500, options?.count ?? 30));
      const maxHoursAhead = Math.max(
        1,
        Math.min(72, options?.maxHoursAhead ?? 24),
      );
      const imminentRatio = Math.max(
        0.25,
        Math.min(1, options?.imminentRatio ?? 0.5),
      );
      const imminentTarget = Math.ceil(count * imminentRatio);
      const includeAdjacent = options?.includeAdjacent ?? false;
      const clearExisting = options?.clearExisting ?? true;
      const includeAnsiblex = options?.includeAnsiblex ?? true;
      const disableJumpbridges = options?.disableJumpbridges ?? true;

      const scoped = resolveScopedSystems(includeAdjacent);
      if (!scoped.ok) {
        const result: FakeTimersResult = {
          created: 0,
          regions: scoped.result.regions,
          adjacentRegion: scoped.result.adjacentRegion,
          ansiblexSystems: [],
          disabledJumpbridges: 0,
        };
        if (result.regions.length === 0) {
          console.warn(
            "[sentinelDebug] no loaded regions; load at least one region first",
            result,
          );
        } else {
          console.warn(
            "[sentinelDebug] no loaded systems for selected region scope",
            result,
          );
        }
        return result;
      }

      const { mapState, scopedSystems, scopeInfo } = scoped;
      const nowMs = Date.now();
      const severityPool = ["low", "medium", "high", "critical"] as const;
      const standingPool = [
        "ours",
        "friendly",
        "neutral",
        "complicated",
        "hostile",
      ] as const;
      const timerKindPool = includeAnsiblex
        ? ["reinforcement", "anchoring", "moon_ore", "skyhook", "custom"]
        : ["reinforcement", "anchoring", "moon_ore", "skyhook", "custom"];
      const structurePool = includeAnsiblex
        ? [
            "upwell_citadel_astrahus",
            "upwell_citadel_fortizar",
            "upwell_refinery_athanor",
            "upwell_refinery_tatara",
            "orbital_skyhook",
            "customs_office_poco",
            "metenox_moon_drill",
            "ansiblex_jump_bridge",
            "custom",
          ]
        : [
            "upwell_citadel_astrahus",
            "upwell_citadel_fortizar",
            "upwell_refinery_athanor",
            "upwell_refinery_tatara",
            "orbital_skyhook",
            "customs_office_poco",
            "metenox_moon_drill",
            "custom",
          ];
      const stagePool = [
        "armor",
        "structure",
        "initial_vulnerability",
        "anchoring",
        "unanchoring",
        "not_applicable",
      ] as const;
      const severityRank: Record<string, number> = {
        low: 1,
        medium: 2,
        high: 3,
        critical: 4,
      };

      const generatedSignals: Record<number, FakeTimerSignal> = {};
      let imminentAssigned = 0;
      const imminentSystems = shuffled(scopedSystems);
      for (let i = 0; i < count; i++) {
        const severity = pickRandom(severityPool);
        const standing = pickRandom(standingPool);
        const timerKind = pickRandom(timerKindPool);
        const structureType = pickRandom(structurePool);
        const stageLabel = pickRandom(stagePool);
        const remainingSlots = count - i;
        const remainingImminent = imminentTarget - imminentAssigned;
        const imminent =
          remainingImminent >= remainingSlots ||
          (remainingImminent > 0 && Math.random() < imminentRatio);
        const system = imminent
          ? imminentSystems[imminentAssigned % imminentSystems.length]
          : pickRandom(scopedSystems);
        if (imminent) {
          imminentAssigned++;
        }
        const expiresAt = imminent
          ? new Date(
              nowMs + 30_000 + Math.floor(Math.random() * 30 * 60 * 1000),
            ).toISOString()
          : new Date(
              nowMs +
                30 * 60 * 1000 +
                Math.floor(Math.random() * maxHoursAhead * 60 * 60 * 1000),
            ).toISOString();
        const celestial = buildDebugCelestial(structureType, system.name);
        const skyhookFullnessPct =
          structureType === "orbital_skyhook"
            ? 5 + Math.floor(Math.random() * 96)
            : undefined;
        const preview: FakeTimerPreview = {
          title: buildDebugTimerTitle(),
          next_expires_at: expiresAt,
          severity,
          standing_type: standing,
          timer_kind: timerKind,
          structure_type: structureType,
          stage_label: stageLabel,
          planet_name: celestial.planet_name,
          moon_name: celestial.moon_name,
          skyhook_fullness_pct: skyhookFullnessPct,
        };

        const current = generatedSignals[system.system];
        if (!current) {
          generatedSignals[system.system] = {
            system_id: system.system,
            count: 1,
            remaining_count: 0,
            next_expires_at: expiresAt,
            severity,
            standing_type: standing,
            timer_kind: timerKind,
            structure_type: structureType,
            stage_label: stageLabel,
            title: preview.title,
            skyhook_fullness_pct: skyhookFullnessPct,
            timers: [preview],
          };
          continue;
        }

        current.count += 1;
        insertTimerPreviewByExpiry(current.timers, preview, 3);
        current.remaining_count = Math.max(
          0,
          current.count - current.timers.length,
        );
        if (Date.parse(expiresAt) < Date.parse(current.next_expires_at)) {
          current.next_expires_at = expiresAt;
          current.structure_type = structureType;
          current.stage_label = stageLabel;
          current.title = preview.title;
          current.timer_kind = timerKind;
          current.standing_type = standing;
          current.skyhook_fullness_pct = skyhookFullnessPct;
        }
        if (severityRank[severity] > (severityRank[current.severity] ?? 0)) {
          current.severity = severity;
        }
      }

      const nextTimerSignals = clearExisting
        ? generatedSignals
        : { ...mapState.timerSignals, ...generatedSignals };
      const ansiblexSystems = new Set(
        Object.values(nextTimerSignals)
          .filter((signal) => signal.timer_kind === "ansiblex_jump_bridge")
          .map((signal) => signal.system_id),
      );

      let disabledJumpbridges = 0;
      const nextJumpbridges = disableJumpbridges
        ? mapState.jumpbridges.map((jumpbridge) => {
            const shouldDisable =
              ansiblexSystems.has(jumpbridge.from) ||
              ansiblexSystems.has(jumpbridge.to);
            const nextDisabled = clearExisting
              ? shouldDisable
              : Boolean(jumpbridge.disabled || shouldDisable);
            if (nextDisabled) {
              disabledJumpbridges++;
            }
            if (nextDisabled === Boolean(jumpbridge.disabled)) {
              return jumpbridge;
            }
            return { ...jumpbridge, disabled: nextDisabled };
          })
        : mapState.jumpbridges;

      useMapStore.setState({
        timerSignals: nextTimerSignals,
        jumpbridges: nextJumpbridges,
      });

      const result: FakeTimersResult = {
        created: Object.keys(generatedSignals).length,
        regions: scopeInfo.regions,
        adjacentRegion: scopeInfo.adjacentRegion,
        ansiblexSystems: Array.from(ansiblexSystems),
        disabledJumpbridges,
      };
      console.info("[sentinelDebug] generated fake timer signals", result);
      return result;
    };

    const generateFakeIntelAndTimerScenario = (
      options?: FakeScenarioOptions,
    ) => {
      const intel = generateFakeIntelReports(options?.intel);
      const timers = generateFakeTimerSignals(options?.timers);
      const result = { intel, timers };
      console.info(
        "[sentinelDebug] generated fake intel + timer scenario",
        result,
      );
      return result;
    };

    api.generateFakeIntelReports = generateFakeIntelReports;
    api.generateFakeTimerSignals = generateFakeTimerSignals;
    api.generateFakeIntelAndTimerScenario = generateFakeIntelAndTimerScenario;
    win.sentinelDebug = api;

    return () => {
      const current = win.sentinelDebug;
      if (current?.generateFakeIntelReports === generateFakeIntelReports) {
        delete current.generateFakeIntelReports;
      }
      if (current?.generateFakeTimerSignals === generateFakeTimerSignals) {
        delete current.generateFakeTimerSignals;
      }
      if (
        current?.generateFakeIntelAndTimerScenario ===
        generateFakeIntelAndTimerScenario
      ) {
        delete current.generateFakeIntelAndTimerScenario;
      }
    };
  }, []);
}

function buildDebugTimerTitle(): string {
  const prefixes = [
    "Abyssal",
    "Triglavian",
    "Drifter",
    "Sleeper",
    "Sansha",
    "Quafe",
    "Angel",
    "Guristas",
    "Concord",
    "Jove",
    "Pochven",
    "Shattered",
  ];
  const cores = [
    "Spindle",
    "Lattice",
    "Relay",
    "Bloom",
    "Chorus",
    "Riddle",
    "Echo",
    "Gyre",
    "Prism",
    "Cipher",
    "Mandala",
    "Mirage",
  ];
  const suffixes = [
    "Protocol",
    "Directive",
    "Signal",
    "Mandate",
    "Thread",
    "Vector",
    "Drift",
    "Convergence",
    "Incident",
    "Sequence",
    "Phase",
    "Whisper",
  ];

  const pick = <T>(items: readonly T[]): T =>
    items[Math.floor(Math.random() * items.length)];
  return `${pick(prefixes)} ${pick(cores)} ${pick(suffixes)}`;
}

function buildDebugCelestial(
  structureType: string,
  systemName: string,
): {
  planet_name?: string;
  moon_name?: string;
} {
  const planetRoll = 1 + Math.floor(Math.random() * 8);
  const moonRoll = 1 + Math.floor(Math.random() * 30);
  if (
    structureType === "orbital_skyhook" ||
    structureType === "customs_office_poco"
  ) {
    return { planet_name: `${systemName} ${planetRoll}` };
  }
  if (
    structureType === "upwell_refinery_athanor" ||
    structureType === "upwell_refinery_tatara" ||
    structureType === "metenox_moon_drill"
  ) {
    return { moon_name: `${systemName} ${planetRoll}-${moonRoll}` };
  }
  return {};
}

function insertTimerPreviewByExpiry(
  previews: FakeTimerPreview[],
  preview: FakeTimerPreview,
  limit: number,
) {
  previews.push(preview);
  previews.sort(
    (a, b) => Date.parse(a.next_expires_at) - Date.parse(b.next_expires_at),
  );
  if (previews.length > limit) {
    previews.length = limit;
  }
}

function shuffled<T>(items: readonly T[]): T[] {
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}
