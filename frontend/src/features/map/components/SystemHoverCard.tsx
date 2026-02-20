import { useLayoutEffect, useRef, useState } from "react";
import { CircleHelp, Droplets, Globe2, Moon } from "lucide-react";
import { createPortal } from "react-dom";
import {
  countdownTone,
  formatCountdown,
  formatStageLabel,
  formatStanding,
  formatStructureType,
  formatTimerDateParts,
  severityBadgeClass,
  stageBadgeClass,
  standingBadgeClass,
  structureBadgeClassByType,
  structureByValue,
  timerKindLabels,
  hostilityRowToneClass,
} from "../../timers";
import type { TimerStageLabel, TimerStandingType } from "../../timers/types";
import type { SystemCharacterBadge } from "../hooks/useSystemCharacters";
import type { TimerSignal, TimerSignalPreview } from "../types";

type SystemHoverCardProps = {
  open: boolean;
  anchor: { x: number; y: number };
  systemName: string;
  characters: SystemCharacterBadge[];
  timerSignal?: TimerSignal;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
};

export default function SystemHoverCard({
  open,
  anchor,
  systemName,
  characters,
  timerSignal,
  onMouseEnter,
  onMouseLeave,
}: SystemHoverCardProps) {
  if (!open || typeof document === "undefined") return null;

  const nowMs = Date.now();
  const use24Hour = readTimerUse24Hour();
  const timerPreviews = buildTimerPreviews(timerSignal);
  const hasPilotsSection = characters.length > 0;
  const hasTimersSection = timerPreviews.length > 0;
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState({
    top: anchor.y + 10,
    left: anchor.x + 10,
  });

  if (!hasPilotsSection && !hasTimersSection) {
    return null;
  }

  useLayoutEffect(() => {
    if (!open) return;

    const updatePosition = () => {
      const card = cardRef.current;
      if (!card) return;
      const margin = 10;
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;
      const cardWidth = card.offsetWidth;
      const cardHeight = card.offsetHeight;

      let left = anchor.x + 10;
      let top = anchor.y + 10;

      if (left + cardWidth > viewportWidth - margin) {
        left = anchor.x - cardWidth - 10;
      }
      if (top + cardHeight > viewportHeight - margin) {
        top = anchor.y - cardHeight - 10;
      }

      left = Math.max(
        margin,
        Math.min(left, viewportWidth - cardWidth - margin),
      );
      top = Math.max(
        margin,
        Math.min(top, viewportHeight - cardHeight - margin),
      );

      setPosition({ top, left });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [
    open,
    anchor.x,
    anchor.y,
    hasPilotsSection,
    hasTimersSection,
    timerPreviews.length,
  ]);

  return createPortal(
    <div
      ref={cardRef}
      className="map-system-hover-card"
      style={{ top: position.top, left: position.left }}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
    >
      <div className="map-system-hover-title">{systemName}</div>

      {hasPilotsSection ? (
        <section className="map-system-hover-section">
          <div className="map-system-hover-section-label">Pilots In System</div>
          <div className="map-system-hover-character-list">
            {characters.map((char) => (
              <div key={char.id} className="map-system-hover-character-row">
                <span
                  className={[
                    "map-system-hover-character-dot",
                    char.inSpace
                      ? "map-system-hover-character-dot-undocked"
                      : "map-system-hover-character-dot-docked",
                  ].join(" ")}
                />
                <span className="map-system-hover-character-name">
                  {char.name}
                </span>
                <span className="map-system-hover-character-state">
                  {char.inSpace ? "Undocked" : "Docked"}
                </span>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {hasTimersSection ? (
        <section className="map-system-hover-section">
          <div className="map-system-hover-section-label">Timers</div>
          <>
            {timerPreviews.map((timer, index) => (
              <TimerEntry
                key={`${timer.next_expires_at}-${timer.title ?? "timer"}-${index}`}
                timer={timer}
                systemName={systemName}
                nowMs={nowMs}
                use24Hour={use24Hour}
              />
            ))}
            {Number(timerSignal?.remaining_count ?? 0) > 0 ? (
              <div className="map-system-hover-more-hint">
                {`${timerSignal?.remaining_count} more timers, see Timers page`}
              </div>
            ) : null}
          </>
        </section>
      ) : null}
    </div>,
    document.body,
  );
}

function TimerEntry({
  timer,
  systemName,
  nowMs,
  use24Hour,
}: {
  timer: TimerSignalPreview;
  systemName: string;
  nowMs: number;
  use24Hour: boolean;
}) {
  const countdownClass = countdownTone(timer.next_expires_at, nowMs);
  const timestamp = formatTimerDateParts(timer.next_expires_at, use24Hour);
  const stageLabel = (timer.stage_label ?? "not_applicable") as TimerStageLabel;
  const title = compactTimerTitle(timer.title, systemName, timer);
  const structureType = timer.structure_type ?? "custom";
  const structure = structureByValue.get(structureType as never);
  const StructureIcon = structure?.icon ?? CircleHelp;

  return (
    <div
      className={`map-system-hover-timer-panel ${hostilityRowToneClass(timer.standing_type)}`}
    >
      <div className="map-system-hover-timer-head">
        <div className="map-system-hover-timer-title">{title}</div>
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
          {formatStanding(timer.standing_type as TimerStandingType)}
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
    </div>
  );
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

function buildTimerPreviews(timerSignal?: TimerSignal): TimerSignalPreview[] {
  if (!timerSignal) return [];
  const candidates: TimerSignalPreview[] = [];
  if (Array.isArray(timerSignal.timers) && timerSignal.timers.length > 0) {
    candidates.push(...timerSignal.timers);
  }
  if (timerSignal.next_expires_at) {
    candidates.push({
      title: timerSignal.title,
      next_expires_at: timerSignal.next_expires_at,
      severity: timerSignal.severity,
      standing_type: timerSignal.standing_type,
      timer_kind: timerSignal.timer_kind,
      structure_type: timerSignal.structure_type,
      stage_label: timerSignal.stage_label,
      planet_name: timerSignal.planet_name,
      moon_name: timerSignal.moon_name,
      skyhook_fullness_pct: timerSignal.skyhook_fullness_pct,
    });
  }
  const deduped = new Map<string, TimerSignalPreview>();
  for (const timer of candidates) {
    const key = `${timer.next_expires_at}|${timer.title ?? ""}|${timer.timer_kind}|${timer.structure_type ?? ""}|${timer.skyhook_fullness_pct ?? ""}`;
    if (!deduped.has(key)) {
      deduped.set(key, timer);
    }
  }
  return Array.from(deduped.values())
    .sort(
      (a, b) => Date.parse(a.next_expires_at) - Date.parse(b.next_expires_at),
    )
    .slice(0, 3);
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

function readTimerUse24Hour(): boolean {
  if (typeof window === "undefined") return true;
  const stored = window.localStorage.getItem("timers:use24Hour");
  return stored ? stored === "true" : true;
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
