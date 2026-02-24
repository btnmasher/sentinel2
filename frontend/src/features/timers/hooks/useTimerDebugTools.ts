import { useEffect } from "react";
import { useMapStore } from "@/features/map";
import {
  contextOptionsFor,
  structureOptions,
  timerFieldsFromContextSelection,
  timerKindLabels,
} from "../config/timerOptions";
import { useTimersStore } from "../store/useTimersStore";
import {
  TimerKind,
  TimerReplacementAction,
  TimerSeverity,
  TimerStageLabel,
  TimerStandingType,
  TimerStatus,
  TimerStructureType,
  type TimerRecord,
} from "../types";

type DebugTimerOptions = {
  count?: number;
  clearExisting?: boolean;
  includeCanceled?: boolean;
  includeExpired?: boolean;
};

type DebugTimerResult = {
  created: number;
  systems: number;
  usedLoadedRegions: boolean;
};

type DebugAPI = {
  generateFakeBoardTimers?: (options?: DebugTimerOptions) => DebugTimerResult;
  [key: string]: unknown;
};

type DebugSystem = {
  id: number;
  name: string;
  regionId: number;
  regionName: string;
};

const fallbackDebugSystems: DebugSystem[] = [
  { id: 30000142, name: "Jita", regionId: 10000002, regionName: "The Forge" },
  {
    id: 30002187,
    name: "Amarr",
    regionId: 10000043,
    regionName: "Domain",
  },
  {
    id: 30002659,
    name: "Dodixie",
    regionId: 10000032,
    regionName: "Sinq Laison",
  },
  {
    id: 30045339,
    name: "1DQ1-A",
    regionId: 10000060,
    regionName: "Delve",
  },
  {
    id: 30004759,
    name: "Rens",
    regionId: 10000030,
    regionName: "Heimatar",
  },
];

const standingPool = [
  TimerStandingType.Ours,
  TimerStandingType.Friendly,
  TimerStandingType.Neutral,
  TimerStandingType.Complicated,
  TimerStandingType.Hostile,
] as const;

const defenderStandingScenarios = [...standingPool] as const;
const attackerStandingScenarios: readonly (TimerStandingType | null)[] = [
  TimerStandingType.Ours,
  TimerStandingType.Friendly,
  TimerStandingType.Neutral,
  TimerStandingType.Complicated,
  TimerStandingType.Hostile,
  null,
];

const severityPool = [
  TimerSeverity.Low,
  TimerSeverity.Medium,
  TimerSeverity.High,
  TimerSeverity.Critical,
] as const;

type DebugCorp = {
  id: number;
  name: string;
  ticker: string;
};

type DebugAlliance = {
  id: number;
  name: string;
  ticker: string;
};

const alliancesByStanding: Record<TimerStandingType, readonly DebugAlliance[]> =
  {
    [TimerStandingType.Ours]: [
      { id: 498125261, name: "Test Alliance Please Ignore", ticker: "TEST" },
    ],
    [TimerStandingType.Friendly]: [
      { id: 99003581, name: "Fraternity.", ticker: "FRT" },
    ],
    [TimerStandingType.Neutral]: [
      { id: 99003995, name: "Invidia Gloriae Comes", ticker: "IGC" },
    ],
    [TimerStandingType.Complicated]: [
      { id: 99012982, name: "OnlyFleets.", ticker: "SL4GS" },
    ],
    [TimerStandingType.Hostile]: [
      { id: 1354830081, name: "Goonswarm Federation", ticker: "CONDI" },
      { id: 1900696668, name: "The Initiative.", ticker: "INIT." },
    ],
  };

const unalignedAlliances: readonly DebugAlliance[] = [
  { id: 99010281, name: "GameTheory", ticker: "GAME" },
];

const corpsByStanding: Record<TimerStandingType, readonly DebugCorp[]> = {
  [TimerStandingType.Ours]: [
    { id: 1018389948, name: "Dreddit", ticker: "B0RT" },
    {
      id: 98638459,
      name: "TEST Infrastructure as a Service",
      ticker: "TIAAS",
    },
    { id: 416584095, name: "Upvote", ticker: "UPVOT" },
  ],
  [TimerStandingType.Friendly]: [
    { id: 98598862, name: "Chaos Arbiter", ticker: "CACX" },
    { id: 98241771, name: "Fraternity Holding", ticker: "FRTHG" },
    {
      id: 98599770,
      name: "Fraternity Building Management",
      ticker: "FRTBM",
    },
  ],
  [TimerStandingType.Neutral]: [
    { id: 98742946, name: "Zhuque squad", ticker: "KR.S" },
    { id: 98727557, name: "GentlemanClub", ticker: "WGCLW" },
    {
      id: 98399796,
      name: "Invidia Administrative",
      ticker: "ID-AD",
    },
  ],
  [TimerStandingType.Complicated]: [
    { id: 98762479, name: "Flat Earth Defense Force", ticker: "F14T" },
    { id: 98252708, name: "Balls Deep Inc.", ticker: "-DRY-" },
    { id: 98469412, name: "Godless Horizon.", ticker: "LOST." },
  ],
  [TimerStandingType.Hostile]: [
    { id: 98370861, name: "KarmaFleet", ticker: "SNOOO" },
    { id: 667531913, name: "GoonWaffe", ticker: "GEWNS" },
    { id: 1344654522, name: "DJ's Retirement Fund", ticker: ".FART" },
    { id: 98534707, name: "Soul Machines", ticker: "C7AM" },
    { id: 98210135, name: "Infinite Point", ticker: "3.14-" },
    { id: 1639878825, name: "Initiative Holding", ticker: "-I.H-" },
  ],
};

export default function useTimerDebugTools() {
  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

    const win = window as Window & { sentinelDebug?: DebugAPI };
    const api = win.sentinelDebug ?? {};

    const generateFakeBoardTimers = (
      options?: DebugTimerOptions,
    ): DebugTimerResult => {
      const count = clamp(options?.count ?? 48, 12, 300);
      const clearExisting = options?.clearExisting ?? true;
      const includeCanceled = options?.includeCanceled ?? true;
      const includeExpired = options?.includeExpired ?? true;

      const scoped = resolveDebugSystems();
      if (scoped.systems.length === 0) {
        const empty: DebugTimerResult = {
          created: 0,
          systems: 0,
          usedLoadedRegions: false,
        };
        console.warn(
          "[sentinelDebug] no systems available for timer debug data",
          empty,
        );
        return empty;
      }

      const now = Date.now();
      const batch = now.toString(36);
      const generated: TimerRecord[] = [];

      // Always include one timer for every valid defender/attacker hostility pairing.
      generated.push(
        ...makeSovereigntyIhubScenarioTimers(scoped.systems, now, batch),
      );

      // Include skyhook reinforcement and extraction to validate category/tab splits.
      generated.push(
        makeSkyhookReinforcementTimer(scoped.systems, now, batch, 3),
        makeSkyhookExtractionTimer(scoped.systems, now, batch, 4),
      );

      for (let i = generated.length; i < count; i++) {
        generated.push(
          makeGenericValidTimer(scoped.systems, now, batch, i, {
            includeCanceled,
            includeExpired,
          }),
        );
      }

      const existing = clearExisting ? [] : useTimersStore.getState().timers;
      useTimersStore.setState({
        timers: [...generated, ...existing],
        loadedAt: Date.now(),
      });

      const result: DebugTimerResult = {
        created: generated.length,
        systems: scoped.systems.length,
        usedLoadedRegions: scoped.usedLoadedRegions,
      };
      console.info(
        "[sentinelDebug] generated fake timer board records",
        result,
      );
      return result;
    };

    api.generateFakeBoardTimers = generateFakeBoardTimers;
    win.sentinelDebug = api;

    return () => {
      const current = win.sentinelDebug;
      if (current?.generateFakeBoardTimers === generateFakeBoardTimers) {
        delete current.generateFakeBoardTimers;
      }
    };
  }, []);
}

function makeSovereigntyIhubScenarioTimers(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
): TimerRecord[] {
  const timers: TimerRecord[] = [];
  let idx = 0;
  for (const defenderStanding of defenderStandingScenarios) {
    for (const attackerStanding of attackerStandingScenarios) {
      if (!isValidSovScenario(defenderStanding, attackerStanding)) {
        continue;
      }
      const timer = makeSovereigntyIhubScenarioTimer(
        systems,
        nowMs,
        batch,
        idx,
        defenderStanding,
        attackerStanding,
      );
      if (!timer) {
        continue;
      }
      timers.push(timer);
      idx++;
    }
  }
  const mixed = makeSovereigntyIhubMixedScenarioTimers(
    systems,
    nowMs,
    batch,
    idx,
  );
  timers.push(...mixed);
  return timers;
}

function makeSovereigntyIhubScenarioTimer(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  idx: number,
  defenderStanding: TimerStandingType,
  attackerStanding: TimerStandingType | null,
): TimerRecord | null {
  const system = systems[idx % systems.length];
  const ownerAlliance = pickScenarioDefenderAlliance(defenderStanding, idx);
  if (!ownerAlliance) {
    return null;
  }
  const attackerAlliance = pickScenarioAttackerAlliance(
    attackerStanding,
    ownerAlliance.id,
    idx,
  );
  if (!attackerAlliance) {
    return null;
  }
  const attackers = clamp(35 + ((idx * 11) % 45), 0, 100);
  const defenders = clamp(100 - attackers, 0, 100);
  const severity =
    defenderStanding === TimerStandingType.Ours
      ? TimerSeverity.Critical
      : defenderStanding === TimerStandingType.Friendly
        ? TimerSeverity.High
        : TimerSeverity.Medium;
  const attackerLabel = formatAttackerLabel(attackerAlliance, attackerStanding);
  return makeBaseTimer({
    id: `debug-sov-${batch}-${idx}`,
    nowMs,
    system,
    standing: defenderStanding,
    severity,
    structureType: TimerStructureType.SovereigntyHub,
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Reinforcement,
    stage: attackers,
    totalStages: 100,
    attackersScorePct: attackers,
    defenderScorePct: defenders,
    replacementAction: TimerReplacementAction.NotReplaceable,
    title: `${system.name} iHub Defense`,
    source: "esi",
    sourceRef: `esi:sovereignty_campaign:${batch}:${idx}`,
    notes: `Attackers: ${attackerLabel}`,
    expiresAt: new Date(nowMs + (20 + idx * 7) * 60 * 1000).toISOString(),
    ownerAlliance,
    ownerCorp: pickCorpForStanding(defenderStanding),
  });
}

function makeSovereigntyIhubMixedScenarioTimers(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  startIdx: number,
): TimerRecord[] {
  const scenarios: ReadonlyArray<{
    defenderStanding: TimerStandingType;
    attackerStandings: ReadonlyArray<TimerStandingType | null>;
  }> = [
    {
      defenderStanding: TimerStandingType.Hostile,
      attackerStandings: [
        TimerStandingType.Friendly,
        TimerStandingType.Neutral,
      ],
    },
    {
      defenderStanding: TimerStandingType.Hostile,
      attackerStandings: [TimerStandingType.Hostile, null],
    },
    {
      defenderStanding: TimerStandingType.Hostile,
      attackerStandings: [
        TimerStandingType.Ours,
        TimerStandingType.Complicated,
        TimerStandingType.Hostile,
      ],
    },
    {
      defenderStanding: TimerStandingType.Neutral,
      attackerStandings: [
        TimerStandingType.Hostile,
        TimerStandingType.Complicated,
      ],
    },
    {
      defenderStanding: TimerStandingType.Complicated,
      attackerStandings: [TimerStandingType.Ours, TimerStandingType.Neutral],
    },
    {
      defenderStanding: TimerStandingType.Ours,
      attackerStandings: [
        TimerStandingType.Hostile,
        TimerStandingType.Neutral,
        null,
      ],
    },
    {
      defenderStanding: TimerStandingType.Friendly,
      attackerStandings: [
        TimerStandingType.Complicated,
        TimerStandingType.Hostile,
      ],
    },
  ];

  const timers: TimerRecord[] = [];
  let idx = startIdx;
  for (const scenario of scenarios) {
    const timer = makeSovereigntyIhubMixedScenarioTimer(
      systems,
      nowMs,
      batch,
      idx,
      scenario.defenderStanding,
      scenario.attackerStandings,
    );
    if (!timer) {
      continue;
    }
    timers.push(timer);
    idx++;
  }
  return timers;
}

function makeSovereigntyIhubMixedScenarioTimer(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  idx: number,
  defenderStanding: TimerStandingType,
  attackerStandings: ReadonlyArray<TimerStandingType | null>,
): TimerRecord | null {
  const uniqueAttackerStandings = [...new Set(attackerStandings)];
  if (
    uniqueAttackerStandings.length === 0 ||
    uniqueAttackerStandings.some(
      (attackerStanding) =>
        !isValidSovScenario(defenderStanding, attackerStanding),
    )
  ) {
    return null;
  }

  const system = systems[idx % systems.length];
  const ownerAlliance = pickScenarioDefenderAlliance(defenderStanding, idx);
  if (!ownerAlliance) {
    return null;
  }
  const attackers = pickScenarioAttackerAlliances(
    uniqueAttackerStandings,
    ownerAlliance.id,
    idx,
  );
  if (attackers.length !== uniqueAttackerStandings.length) {
    return null;
  }

  const attackersPct = clamp(38 + ((idx * 13) % 42), 0, 100);
  const defendersPct = clamp(100 - attackersPct, 0, 100);
  const severity =
    defenderStanding === TimerStandingType.Ours
      ? TimerSeverity.Critical
      : defenderStanding === TimerStandingType.Friendly
        ? TimerSeverity.High
        : TimerSeverity.Medium;
  const attackerLabel = attackers
    .map(({ alliance, standing }) => formatAttackerLabel(alliance, standing))
    .join(", ");

  return makeBaseTimer({
    id: `debug-sov-mixed-${batch}-${idx}`,
    nowMs,
    system,
    standing: defenderStanding,
    severity,
    structureType: TimerStructureType.SovereigntyHub,
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Reinforcement,
    stage: attackersPct,
    totalStages: 100,
    attackersScorePct: attackersPct,
    defenderScorePct: defendersPct,
    replacementAction: TimerReplacementAction.NotReplaceable,
    title: `${system.name} iHub Defense (Mixed Attackers)`,
    source: "esi",
    sourceRef: `esi:sovereignty_campaign:${batch}:mixed:${idx}`,
    notes: `Attackers: ${attackerLabel}`,
    expiresAt: new Date(nowMs + (27 + idx * 5) * 60 * 1000).toISOString(),
    ownerAlliance,
    ownerCorp: pickCorpForStanding(defenderStanding),
  });
}

function isValidSovScenario(
  defenderStanding: TimerStandingType,
  attackerStanding: TimerStandingType | null,
): boolean {
  // Friendly organizations (ours + friendly) do not attack one another.
  if (
    isFriendlyStanding(defenderStanding) &&
    attackerStanding &&
    isFriendlyStanding(attackerStanding)
  ) {
    return false;
  }
  return true;
}

function isFriendlyStanding(standing: TimerStandingType): boolean {
  return (
    standing === TimerStandingType.Ours ||
    standing === TimerStandingType.Friendly
  );
}

function pickScenarioDefenderAlliance(
  standing: TimerStandingType,
  seed: number,
): DebugAlliance | null {
  const options = alliancesByStanding[standing];
  if (options.length === 0) {
    return null;
  }
  return options[seed % options.length];
}

function pickScenarioAttackerAlliance(
  standing: TimerStandingType | null,
  defenderAllianceID: number,
  seed: number,
  excludedAllianceIds: ReadonlySet<number> = new Set(),
): DebugAlliance | null {
  const options =
    standing === null ? unalignedAlliances : alliancesByStanding[standing];
  const filtered = options.filter(
    (option) =>
      option.id !== defenderAllianceID && !excludedAllianceIds.has(option.id),
  );
  if (filtered.length === 0) {
    return null;
  }
  return filtered[seed % filtered.length];
}

function pickScenarioAttackerAlliances(
  standings: ReadonlyArray<TimerStandingType | null>,
  defenderAllianceID: number,
  seed: number,
): Array<{ alliance: DebugAlliance; standing: TimerStandingType | null }> {
  const used = new Set<number>([defenderAllianceID]);
  const picked: Array<{
    alliance: DebugAlliance;
    standing: TimerStandingType | null;
  }> = [];
  for (let i = 0; i < standings.length; i++) {
    const standing = standings[i];
    const alliance = pickScenarioAttackerAlliance(
      standing,
      defenderAllianceID,
      seed + i,
      used,
    );
    if (!alliance) {
      return picked;
    }
    picked.push({ alliance, standing });
    used.add(alliance.id);
  }
  return picked;
}

function formatAttackerLabel(
  alliance: DebugAlliance,
  standing: TimerStandingType | null,
): string {
  if (!standing) {
    return `[${alliance.ticker}] ${alliance.name}`;
  }
  return `[${alliance.ticker}] ${alliance.name} (${standing})`;
}

function makeSkyhookReinforcementTimer(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  idx: number,
): TimerRecord {
  const system = pickRandom(systems);
  return makeBaseTimer({
    id: `debug-skyhook-reinf-${batch}-${idx}`,
    nowMs,
    system,
    standing: TimerStandingType.Ours,
    severity: TimerSeverity.Critical,
    structureType: TimerStructureType.OrbitalSkyhook,
    timerKind: TimerKind.Reinforcement,
    stageLabel: TimerStageLabel.Reinforcement,
    stage: 1,
    totalStages: 1,
    attackersScorePct: 0,
    defenderScorePct: 100,
    replacementAction: TimerReplacementAction.AllianceReplacement,
    title: `${system.name} Skyhook Reinforcement`,
    source: "esi",
    sourceRef: `esi:notification:${batch}:${idx}`,
    notes: "From SkyhookUnderAttack notification",
    expiresAt: new Date(nowMs + 42 * 60 * 1000).toISOString(),
    ownerCorp: pickCorpForStanding(TimerStandingType.Ours),
    ownerAlliance: pickAllianceForStanding(TimerStandingType.Ours),
    planetName: `${system.name} I`,
  });
}

function makeSkyhookExtractionTimer(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  idx: number,
): TimerRecord {
  const system = pickRandom(systems);
  const standing = pickRandom(standingPool);
  return makeBaseTimer({
    id: `debug-skyhook-extract-${batch}-${idx}`,
    nowMs,
    system,
    standing,
    severity: pickRandom(severityPool),
    structureType: TimerStructureType.OrbitalSkyhook,
    timerKind: TimerKind.Extraction,
    stageLabel: TimerStageLabel.ExtractionWindow,
    stage: 1,
    totalStages: 1,
    attackersScorePct: 0,
    defenderScorePct: 0,
    replacementAction: TimerReplacementAction.LogiReplacement,
    title: `${system.name} Skyhook Extraction`,
    source: "manual",
    sourceRef: `debug:skyhook_extraction:${batch}:${idx}`,
    notes: "Extraction window test timer",
    expiresAt: new Date(nowMs + 95 * 60 * 1000).toISOString(),
    ownerCorp: pickCorpForStanding(standing),
    ownerAlliance: pickAllianceForStanding(standing),
    planetName: `${system.name} II`,
    skyhookFullnessPct: clamp(25 + Math.round(Math.random() * 65), 0, 100),
  });
}

function makeGenericValidTimer(
  systems: DebugSystem[],
  nowMs: number,
  batch: string,
  idx: number,
  options: { includeCanceled: boolean; includeExpired: boolean },
): TimerRecord {
  const system = pickRandom(systems);
  const structureType = pickRandom(
    structureOptions.map((option) => option.value),
  );
  const context = pickRandom(contextOptionsFor(structureType));
  const { timerKind, stageLabel } = timerFieldsFromContextSelection(
    context.value,
  );
  const standing = pickRandom(standingPool);
  const severity = pickRandom(severityPool);
  const replacement = pickRandom([
    TimerReplacementAction.NotReplaceable,
    TimerReplacementAction.LogiReplacement,
    TimerReplacementAction.CorpReplacement,
    TimerReplacementAction.AllianceReplacement,
  ] as const);

  const isExpired = options.includeExpired && idx % 9 === 0;
  const isCanceled = options.includeCanceled && idx % 11 === 0;
  const expiresOffsetMinutes = isExpired
    ? -1 * (5 + (idx % 235))
    : pickFutureOffsetMinutes();
  const expiresAt = new Date(nowMs + expiresOffsetMinutes * 60 * 1000);

  const stage = timerKind === TimerKind.Reinforcement ? 1 + (idx % 3) : 1;
  const totalStages = timerKind === TimerKind.Reinforcement ? 3 : 1;
  const ownerCorp = pickCorpForStanding(standing);
  const ownerAlliance = pickAllianceForStanding(standing);
  const title = `${system.name} ${timerKindLabels[timerKind]}`;

  return makeBaseTimer({
    id: `debug-timer-${batch}-${idx}`,
    nowMs,
    system,
    standing,
    severity,
    structureType,
    timerKind,
    stageLabel,
    stage,
    totalStages,
    attackersScorePct: 0,
    defenderScorePct: 0,
    replacementAction: replacement,
    title,
    source: "manual",
    sourceRef: `debug:timer:${batch}:${idx}`,
    notes: "Synthetic timer for board layout QA",
    expiresAt: expiresAt.toISOString(),
    ownerCorp,
    ownerAlliance,
    planetName: usesPlanet(structureType)
      ? `${system.name} ${(idx % 10) + 1}`
      : "",
    moonName: usesMoon(structureType) ? `${system.name} ${(idx % 6) + 1}` : "",
    status: isCanceled ? TimerStatus.Canceled : TimerStatus.Active,
    canceledAt: isCanceled ? new Date(nowMs).toISOString() : "",
  });
}

function makeBaseTimer(input: {
  id: string;
  nowMs: number;
  system: DebugSystem;
  standing: TimerStandingType;
  timerKind: TimerKind;
  structureType: TimerStructureType;
  stageLabel: TimerStageLabel;
  severity: TimerSeverity;
  status?: TimerStatus;
  expiresAt: string;
  source: string;
  sourceRef: string;
  title: string;
  notes: string;
  replacementAction: TimerReplacementAction;
  stage: number;
  totalStages: number;
  attackersScorePct: number;
  defenderScorePct: number;
  ownerCorp?: { id: number; name: string; ticker: string };
  ownerAlliance?: { id: number; name: string; ticker: string };
  skyhookFullnessPct?: number;
  planetName?: string;
  moonName?: string;
  canceledAt?: string;
}): TimerRecord {
  const nowISO = new Date(input.nowMs).toISOString();
  const ownerCorp = input.ownerCorp ?? { id: 0, name: "", ticker: "" };
  const ownerAlliance = input.ownerAlliance ?? { id: 0, name: "", ticker: "" };
  return {
    id: input.id,
    title: input.title,
    system_id: input.system.id,
    system_name: input.system.name,
    region_id: input.system.regionId,
    region_name: input.system.regionName,
    standing_type: input.standing,
    timer_kind: input.timerKind,
    structure_type: input.structureType,
    stage_label: input.stageLabel,
    planet_id: input.planetName ? 1 : 0,
    planet_name: input.planetName ?? "",
    moon_id: input.moonName ? 1 : 0,
    moon_name: input.moonName ?? "",
    owner_corporation_id: ownerCorp.id,
    owner_corporation_name: ownerCorp.name,
    owner_corporation_ticker: ownerCorp.ticker,
    owner_alliance_id: ownerAlliance.id,
    owner_alliance_name: ownerAlliance.name,
    owner_alliance_ticker: ownerAlliance.ticker,
    skyhook_fullness_pct: input.skyhookFullnessPct ?? 0,
    attackers_score_pct: input.attackersScorePct,
    defender_score_pct: input.defenderScorePct,
    stage: input.stage,
    total_stages: input.totalStages,
    severity: input.severity,
    status: input.status ?? TimerStatus.Active,
    expires_at: input.expiresAt,
    source: input.source,
    source_ref: input.sourceRef,
    notes: input.notes,
    raw_text: "synthetic timer debug payload",
    replacement_action: input.replacementAction,
    created_by: "",
    created_by_name: input.source === "esi" ? "System" : "Debug User",
    canceled_by: "",
    canceled_at: input.canceledAt ?? "",
    created: nowISO,
    updated: nowISO,
  };
}

function resolveDebugSystems(): {
  systems: DebugSystem[];
  usedLoadedRegions: boolean;
} {
  const state = useMapStore.getState();
  const systems = Object.values(state.systems);
  const regions = state.regions;
  if (systems.length === 0) {
    return { systems: fallbackDebugSystems, usedLoadedRegions: false };
  }

  const loadedRegions = new Set(
    state.mapRegions
      .map((raw) => Number.parseInt(raw, 10))
      .filter((value) => Number.isFinite(value)),
  );
  const scoped =
    loadedRegions.size > 0
      ? systems.filter((system) => loadedRegions.has(system.region))
      : systems;
  const picked = scoped.length > 0 ? scoped : systems;

  return {
    systems: picked.slice(0, 600).map((system) => ({
      id: system.system,
      name: system.name,
      regionId: system.region,
      regionName: regions[system.region]?.name || `Region ${system.region}`,
    })),
    usedLoadedRegions: loadedRegions.size > 0 && scoped.length > 0,
  };
}

function usesPlanet(structureType: TimerStructureType): boolean {
  return (
    structureType === TimerStructureType.OrbitalSkyhook ||
    structureType === TimerStructureType.MercenaryDen ||
    structureType === TimerStructureType.CustomsOfficePoco
  );
}

function usesMoon(structureType: TimerStructureType): boolean {
  return structureType === TimerStructureType.MetenoxMoonDrill;
}

function pickRandom<T>(items: readonly T[]): T {
  return items[Math.floor(Math.random() * items.length)];
}

function pickAllianceForStanding(standing: TimerStandingType): DebugAlliance {
  const options = alliancesByStanding[standing];
  return pickRandom(options);
}

function pickCorpForStanding(standing: TimerStandingType): DebugCorp {
  const options = corpsByStanding[standing];
  return pickRandom(options);
}

function clamp(value: number, minValue: number, maxValue: number): number {
  return Math.max(minValue, Math.min(maxValue, value));
}

function pickFutureOffsetMinutes(): number {
  const roll = Math.random();
  if (roll < 0.25) {
    // 15m - 2h
    return 15 + Math.floor(Math.random() * (120 - 15 + 1));
  }
  if (roll < 0.6) {
    // 2h - 12h
    return 121 + Math.floor(Math.random() * (720 - 121 + 1));
  }
  if (roll < 0.9) {
    // 12h - 72h
    return 721 + Math.floor(Math.random() * (4320 - 721 + 1));
  }
  // 3d - 10d
  return 4321 + Math.floor(Math.random() * (14400 - 4321 + 1));
}
