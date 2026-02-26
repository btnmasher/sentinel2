import { CircleHelp, Droplets, Globe2, Moon } from "lucide-react";
import {
  countdownTone,
  formatCountdown,
  formatStageLabel,
  formatStanding,
  formatStructureType,
  formatTimerDateParts,
  isNeutralTimerStanding,
  severityBadgeClass,
  stageBadgeClass,
  structureBadgeClassByType,
  structureByValue,
  timerKindLabels,
  TimerStageLabel,
} from "../../timers";
import {
  hostilityRowToneClass,
  standingBadgeClass,
  type StandingType,
} from "@/features/shared";
import type { TimerSignalPreview } from "../types";

type SystemContext = {
  systemName: string;
  regionName: string;
  regionLoaded: boolean;
};

export type TimerHoverPanelProps = {
  timer: TimerSignalPreview;
  baseSystemName: string;
  nowMs: number;
  use24Hour: boolean;
  systemContext?: SystemContext;
  onClick?: () => void;
  interactiveClassName?: string;
};

export function readTimerUse24Hour(): boolean {
  if (typeof window === "undefined") return true;
  const stored = window.localStorage.getItem("timers:use24Hour");
  return stored ? stored === "true" : true;
}

export default function TimerHoverPanel({
  timer,
  baseSystemName,
  nowMs,
  use24Hour,
  systemContext,
  onClick,
  interactiveClassName = "timer-count-next-up-card",
}: TimerHoverPanelProps) {
  const countdownClass = countdownTone(timer.next_expires_at, nowMs);
  const timestamp = formatTimerDateParts(timer.next_expires_at, use24Hour);
  const stageLabel = (timer.stage_label ??
    TimerStageLabel.NotApplicable) as TimerStageLabel;
  const title = compactTimerTitle(timer.title, baseSystemName, timer);
  const structureType = timer.structure_type ?? "custom";
  const isNeutralHostility = isNeutralTimerStanding(timer.standing_type ?? "");
  const structure = structureByValue.get(structureType as never);
  const StructureIcon = structure?.icon ?? CircleHelp;

  const panelClassName =
    `map-system-hover-timer-panel ${hostilityRowToneClass(timer.standing_type)} ${
      isNeutralHostility ? "map-system-hover-timer-panel-neutral" : ""
    } ${onClick ? interactiveClassName : ""}`.trim();

  const content = (
    <>
      <div className="map-system-hover-timer-head">
        <div className="map-system-hover-timer-title">
          {systemContext ? (
            <div className="space-y-0.5">
              <div
                className={`text-xs ${
                  systemContext.regionLoaded
                    ? "report-item-system-link"
                    : "report-item-unloaded-region"
                }`}
              >
                {systemContext.systemName}
                <span className="report-item-unloaded-region">
                  {" "}
                  &gt; {systemContext.regionName}
                </span>
              </div>
              <div>{title}</div>
            </div>
          ) : (
            title
          )}
        </div>
        <span
          className={`badge timer-row-badge map-system-hover-timer-priority ${severityBadgeClass(timer.severity)}`}
        >
          {humanize(timer.severity)}
        </span>
      </div>
      <div className={`map-system-hover-timer-countdown ${countdownClass}`}>
        {formatCountdown(timer.next_expires_at, nowMs)}
      </div>
      <div className="map-system-hover-static-row">
        {timestamp ? (
          <>
            <StyledDateText parts={timestamp.local} />
            <span className="map-system-hover-static-paren">(</span>
            <StyledDateText
              parts={{
                ...timestamp.eve,
                timezone: "EVE TIME",
              }}
            />
            <span className="map-system-hover-static-paren">)</span>
          </>
        ) : (
          <span className="map-system-hover-fallback-time">
            {timer.next_expires_at}
          </span>
        )}
      </div>
      <div className="map-system-hover-timer-badges">
        <span
          className={`badge timer-row-badge ${standingBadgeClass(timer.standing_type)}`}
        >
          {formatStanding(timer.standing_type as StandingType)}
        </span>
        <span
          className={`badge timer-row-badge ${structureBadgeClassByType(structureType)}`}
        >
          <StructureIcon className="h-3 w-3" />{" "}
          {formatStructureType(structureType)}
        </span>
        {showSkyhookFullnessBadge(timer) ? (
          <span className="badge timer-row-badge timer-skyhook-fullness-badge">
            <Droplets className="h-3 w-3" />{" "}
            {Math.round(Number(timer.skyhook_fullness_pct))}% Full
          </span>
        ) : null}
        {stageLabel !== "not_applicable" ? (
          <span
            className={`badge timer-row-badge ${stageBadgeClass(timer.stage_label)}`}
          >
            {formatStageLabel(stageLabel)}
          </span>
        ) : null}
      </div>
      <TimerCelestialMeta
        planetName={timer.planet_name}
        moonName={timer.moon_name}
      />
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={`${panelClassName} w-full text-left`}
      >
        {content}
      </button>
    );
  }

  return <div className={panelClassName}>{content}</div>;
}

function TimerCelestialMeta({
  planetName,
  moonName,
}: {
  planetName?: string;
  moonName?: string;
}) {
  if (moonName) {
    return (
      <div className="map-system-hover-timer-meta">
        <span className="map-system-hover-timer-meta-chip">
          <Moon className="h-3 w-3" />
          <span className="map-system-hover-timer-meta-label">Moon</span>
        </span>
        <span className="map-system-hover-timer-meta-value">{moonName}</span>
      </div>
    );
  }

  if (planetName) {
    return (
      <div className="map-system-hover-timer-meta">
        <span className="map-system-hover-timer-meta-chip">
          <Globe2 className="h-3 w-3" />
          <span className="map-system-hover-timer-meta-label">Planet</span>
        </span>
        <span className="map-system-hover-timer-meta-value">{planetName}</span>
      </div>
    );
  }

  return null;
}

function StyledDateText({
  parts,
}: {
  parts: {
    year: string;
    month: string;
    day: string;
    hour: string;
    minute: string;
    second: string;
    suffix?: string;
    timezone?: string;
  };
}) {
  return (
    <span className="timer-styled-date">
      <span>{parts.year}</span>
      <span className="text-success/85">-</span>
      <span>{parts.month}</span>
      <span className="text-success/85">-</span>
      <span>{parts.day}</span>
      <span className="timer-styled-date-divider">|</span>
      <span>{parts.hour}</span>
      <span className="text-success/85">:</span>
      <span>{parts.minute}</span>
      <span className="text-success/85">:</span>
      <span>{parts.second}</span>
      {parts.suffix ? (
        <span className="timer-styled-date-suffix">{parts.suffix}</span>
      ) : null}
      {parts.timezone ? (
        <span className="timer-styled-date-suffix">{parts.timezone}</span>
      ) : null}
    </span>
  );
}

function showSkyhookFullnessBadge(timer: TimerSignalPreview): boolean {
  const isSkyhookExtraction =
    timer.structure_type === "orbital_skyhook" &&
    (timer.timer_kind === "extraction" || timer.timer_kind === "skyhook");
  return (
    isSkyhookExtraction &&
    Number.isFinite(timer.skyhook_fullness_pct) &&
    Number(timer.skyhook_fullness_pct) > 0
  );
}

function timerKindLabel(kind: string): string {
  return (
    timerKindLabels[kind as keyof typeof timerKindLabels] ?? humanize(kind)
  );
}

function compactTimerTitle(
  rawTitle: string | undefined,
  systemName: string,
  timer: TimerSignalPreview,
): string {
  const cleaned = (rawTitle ?? "").trim();
  if (cleaned) {
    const withSeparator = `${systemName} `;
    if (cleaned.startsWith(withSeparator)) {
      const stripped = cleaned.slice(withSeparator.length).trim();
      if (stripped) return stripped;
    }
    if (cleaned === systemName) {
      return fallbackTitle(timer);
    }
    return cleaned;
  }
  return fallbackTitle(timer);
}

function fallbackTitle(timer: TimerSignalPreview): string {
  const structure = formatStructureType(timer.structure_type ?? "custom");
  const kind = timerKindLabel(timer.timer_kind);
  return `${structure} ${kind}`;
}

function humanize(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "Unknown";
  return trimmed
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
