import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { SystemCharacterBadge } from "../hooks/useSystemCharacters";
import type { TimerSignal, TimerSignalPreview } from "../types";
import TimerHoverPanel, { readTimerUse24Hour } from "./TimerHoverPanel";

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
      className="hover-card-surface map-system-hover-card"
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
              <TimerHoverPanel
                key={`${timer.next_expires_at}-${timer.title ?? "timer"}-${index}`}
                timer={timer}
                baseSystemName={systemName}
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
