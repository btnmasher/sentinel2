import { useEffect, useState } from "react";
import { Clock3 } from "lucide-react";
import HoverCard from "@/components/HoverCard";
import { useMapStore } from "@/features/map";
import type { TimerSignal, TimerSignalPreview } from "@/features/map/types";
import TimerHoverPanel, {
  readTimerUse24Hour,
} from "@/features/map/components/TimerHoverPanel";

type Severity = "critical" | "high" | "medium" | "low" | "unknown";
type EnrichedTimerPreview = TimerSignalPreview & { systemId: number };

function severityRank(value: Severity): number {
  switch (value) {
    case "critical":
      return 4;
    case "high":
      return 3;
    case "medium":
      return 2;
    case "low":
      return 1;
    default:
      return 0;
  }
}

function normalizeSeverity(value: string): Severity {
  const cleaned = value.trim().toLowerCase();
  if (
    cleaned === "critical" ||
    cleaned === "high" ||
    cleaned === "medium" ||
    cleaned === "low"
  ) {
    return cleaned;
  }
  return "unknown";
}

function severityDotColor(value: Severity): string {
  switch (value) {
    case "critical":
      return "#ef4444";
    case "high":
      return "#f59e0b";
    case "medium":
      return "#22c55e";
    case "low":
      return "#0ea5e9";
    default:
      return "#64748b";
  }
}

function badgeToneClass(value: Severity): string {
  switch (value) {
    case "critical":
      return "intel-status-text-stale";
    case "high":
      return "intel-status-text-warn";
    case "medium":
      return "intel-status-text-active";
    case "low":
      return "text-sky-300";
    default:
      return "text-slate-400";
  }
}

function buildTimerPreviews(signals: TimerSignal[]) {
  const all: EnrichedTimerPreview[] = [];
  for (const signal of signals) {
    if (Array.isArray(signal.timers) && signal.timers.length > 0) {
      all.push(
        ...signal.timers.map((timer) => ({
          ...timer,
          systemId: signal.system_id,
        })),
      );
      continue;
    }
    if (!signal.next_expires_at) continue;
    all.push({
      systemId: signal.system_id,
      title: signal.title,
      next_expires_at: signal.next_expires_at,
      severity: signal.severity,
      standing_type: signal.standing_type,
      timer_kind: signal.timer_kind,
      structure_type: signal.structure_type,
      stage_label: signal.stage_label,
      planet_name: signal.planet_name,
      moon_name: signal.moon_name,
      skyhook_fullness_pct: signal.skyhook_fullness_pct,
    });
  }
  return all;
}

export default function TimerCountBadge() {
  const [nowMs, setNowMs] = useState(() => Date.now());
  const timerSignals = useMapStore((s) => s.timerSignals);
  const displayTimers = useMapStore((s) => s.displayTimers !== false);
  const systems = useMapStore((s) => s.systems);
  const regions = useMapStore((s) => s.regions);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const signals = Object.values(timerSignals);

  const visibleTimers = displayTimers
    ? signals.reduce((sum, signal) => {
        const count = Number(signal.count);
        return sum + (Number.isFinite(count) && count > 0 ? count : 0);
      }, 0)
    : 0;
  const systemsWithTimers = displayTimers ? signals.length : 0;

  const timerPreviews = displayTimers ? buildTimerPreviews(signals) : [];
  const datedTimers = timerPreviews
    .map((timer) => ({
      timer,
      expiresMs: Date.parse(timer.next_expires_at),
    }))
    .filter((entry) => Number.isFinite(entry.expiresMs))
    .sort((a, b) => a.expiresMs - b.expiresMs);
  const nextUpEntry =
    datedTimers.find((entry) => entry.expiresMs >= nowMs) ?? datedTimers[0];
  const nextUp = nextUpEntry?.timer;

  const nextSeverity = normalizeSeverity(nextUp?.severity ?? "");
  const badgeTone =
    displayTimers && visibleTimers > 0
      ? badgeToneClass(nextSeverity)
      : "intel-status-text-stale";
  const nextUpMs = nextUpEntry?.expiresMs ?? NaN;
  const nextUpImminent =
    Number.isFinite(nextUpMs) &&
    nextUpMs > nowMs &&
    nextUpMs - nowMs <= 30 * 60 * 1000;
  const imminentPulseToneClass = `intel-status-icon--timer-imminent-${nextSeverity}`;

  const severityCounts: Record<Severity, number> = {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    unknown: 0,
  };
  if (displayTimers) {
    for (const signal of signals) {
      if (Array.isArray(signal.timers) && signal.timers.length > 0) {
        for (const timer of signal.timers) {
          severityCounts[normalizeSeverity(timer.severity)] += 1;
        }
      } else {
        const count = Math.max(1, Number(signal.count) || 0);
        severityCounts[normalizeSeverity(signal.severity)] += count;
      }
    }
  }

  const hasNextUp = Boolean(nextUp);
  const nextUpSystem = nextUp ? systems[nextUp.systemId] : undefined;
  const nextUpSystemLabel = nextUpSystem?.name ?? "Unknown System";
  const nextUpRegionId = nextUpSystem ? String(nextUpSystem.region) : "";
  const nextUpRegionLoaded = nextUpRegionId
    ? mapRegions.includes(nextUpRegionId)
    : false;
  const nextUpRegionName =
    nextUpSystem && regions[nextUpSystem.region]?.name
      ? regions[nextUpSystem.region]?.name
      : nextUpSystem
        ? `Region ${nextUpSystem.region}`
        : "Unknown Region";
  const use24Hour = readTimerUse24Hour();

  const focusNextUpSystem = () => {
    if (!nextUp) return;
    if (nextUpRegionId && !mapRegions.includes(nextUpRegionId)) {
      updateMapConfig({ mapRegions: [...mapRegions, nextUpRegionId] });
    }
    setSystemSearch(nextUp.systemId);
  };

  const severityRowsBase: Array<{ key: Severity; label: string }> = [
    { key: "critical", label: "Critical" },
    { key: "high", label: "High" },
    { key: "medium", label: "Medium" },
    { key: "low", label: "Low" },
    { key: "unknown", label: "Unknown" },
  ];
  const severityRows = severityRowsBase.filter(
    (row) => severityCounts[row.key] > 0,
  );

  useEffect(() => {
    const tick = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(tick);
  }, []);

  return (
    <HoverCard
      trigger={
        <button
          type="button"
          className="flex items-center gap-2 rounded-full bg-base-300/70 px-2 py-1 text-base-content transition-colors hover:bg-base-300/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          aria-label="Visible timer count"
        >
          <span
            className={`intel-badge-icon-bg inline-flex h-6 w-6 items-center justify-center rounded-full ${
              nextUpImminent
                ? `intel-status-icon--timer-imminent ${imminentPulseToneClass}`
                : visibleTimers > 0
                  ? ""
                  : "intel-status-icon--alert"
            }`}
          >
            <Clock3 className={`h-3.5 w-3.5 ${badgeTone}`} />
          </span>
          <span>{visibleTimers}</span>
        </button>
      }
      className="hover-card-surface intel-badge-hover-card w-80 p-3 text-xs"
    >
      <p className="text-sm font-semibold">
        {displayTimers ? "Visible timers" : "Timer overlay hidden"}
      </p>
      <p className="mt-1 text-base-content/80">
        {displayTimers
          ? `${visibleTimers} timer${visibleTimers === 1 ? "" : "s"} across ${systemsWithTimers} system${systemsWithTimers === 1 ? "" : "s"}.`
          : "Enable Timers on the top bar to show timer overlays on the map."}
      </p>

      {displayTimers && severityRows.length > 0 ? (
        <div className="mt-2 space-y-1">
          {severityRows
            .sort((a, b) => severityRank(b.key) - severityRank(a.key))
            .map((row) => (
              <div
                key={row.key}
                className="flex items-center justify-between text-[11px]"
              >
                <span className="inline-flex items-center gap-1.5 text-base-content/85">
                  <span
                    className="inline-block h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: severityDotColor(row.key) }}
                  />
                  {row.label}
                </span>
                <span className="font-semibold text-base-content/90">
                  {severityCounts[row.key]}
                </span>
              </div>
            ))}
        </div>
      ) : null}

      {hasNextUp && nextUp ? (
        <div className="mt-3">
          <div className="mb-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-base-content/70">
            Next Up
          </div>
          <TimerHoverPanel
            timer={nextUp}
            baseSystemName={nextUpSystemLabel}
            nowMs={nowMs}
            use24Hour={use24Hour}
            systemContext={{
              systemName: nextUpSystemLabel,
              regionName: nextUpRegionName,
              regionLoaded: nextUpRegionLoaded,
            }}
            onClick={focusNextUpSystem}
            interactiveClassName="timer-count-next-up-card"
          />
        </div>
      ) : null}
    </HoverCard>
  );
}
