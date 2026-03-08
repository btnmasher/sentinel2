import { useRef, useState } from "react";
import type { MouseEvent } from "react";
import { useIntelStore } from "@/features/intel";
import {
  normalizeTimerSeverity,
  timerSeverityDotColor,
  timerSeverityRank,
} from "@/features/timers";
import { useUIStore } from "@/app/store/uiStore";
import { useMapStore } from "../store/mapStore";
import { colorToHex, transformComponent } from "../utils/mapUtils";
import { useSystemRouteState } from "../hooks/useSystemRouteState";
import { useSystemThreatState } from "../hooks/useSystemThreatState";
import { useSystemVisibility } from "../hooks/useSystemVisibility";
import { useSystemCharacters } from "../hooks/useSystemCharacters";
import { useSystemInteractions } from "../hooks/useSystemInteractions";
import { useSystemBorderState } from "../hooks/useSystemBorderState";
import SystemBorderDisplay from "./SystemBorderDisplay";
import SystemHoverCard from "./SystemHoverCard";

export default function MapSystem({
  systemId,
  scale,
}: {
  systemId: number;
  scale: number;
}) {
  const systems = useMapStore((s) => s.systems);
  const timerSignals = useMapStore((s) => s.timerSignals);
  const displayTimers = useMapStore((s) => s.displayTimers !== false);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const { isRouteOrigin, isRouteWaypoint } = useSystemRouteState(systemId);
  const { showSystem, showSystemText } = useSystemVisibility();
  const locatedCharacters = useSystemCharacters(systemId);
  const { handleMouseUp, handleContextMenu } = useSystemInteractions(systemId);
  const systemBorder = useSystemBorderState({
    isRouteOrigin,
    isRouteWaypoint,
    characters: locatedCharacters,
  });

  const logFilters = useIntelStore((s) => s.logFilters);
  const { systemFill, threatStage, alerting, clearFlashing } =
    useSystemThreatState(systemId);

  const system = systems[systemId];
  const timerSignal = displayTimers ? timerSignals[systemId] : undefined;
  const hasContextMenuOpen = useUIStore((s) => Boolean(s.contextMenu));
  const [isSystemHovered, setIsSystemHovered] = useState(false);
  const [isHoverCardHovered, setIsHoverCardHovered] = useState(false);
  const [timerTooltipAnchor, setTimerTooltipAnchor] = useState({ x: 0, y: 0 });
  const hideTimerTooltipTimeoutRef = useRef<number | undefined>(undefined);

  if (!system) return null;

  const hasThreatOverlay = threatStage !== "normal";
  const opacity = jumpranges.enabled ? 1 : hasThreatOverlay ? 0.82 : 0.65;
  const textColor = logFilters.system.includes(systemId)
    ? colorToHex("green lighten-1")
    : "rgba(255, 255, 255, 0.8)";

  const showSystemHoverCard = (event: MouseEvent<SVGRectElement>) => {
    if (hideTimerTooltipTimeoutRef.current) {
      window.clearTimeout(hideTimerTooltipTimeoutRef.current);
      hideTimerTooltipTimeoutRef.current = undefined;
    }
    const rect = event.currentTarget.getBoundingClientRect();
    setTimerTooltipAnchor({ x: rect.right, y: rect.top });
    setIsSystemHovered(true);
  };

  const hideTimerTooltip = () => {
    hideTimerTooltipTimeoutRef.current = window.setTimeout(() => {
      setIsSystemHovered(false);
      hideTimerTooltipTimeoutRef.current = undefined;
    }, 70);
  };

  const handleSystemContextMenu = (event: React.MouseEvent<SVGGElement>) => {
    if (hideTimerTooltipTimeoutRef.current) {
      window.clearTimeout(hideTimerTooltipTimeoutRef.current);
      hideTimerTooltipTimeoutRef.current = undefined;
    }
    setIsSystemHovered(false);
    setIsHoverCardHovered(false);
    handleContextMenu(event);
  };

  const timerTooltipOpen =
    !hasContextMenuOpen && (isSystemHovered || isHoverCardHovered);

  return (
    <g
      className="map-system"
      transform={transformComponent(system.position, scale)}
      style={{ display: showSystem ? "block" : "none" }}
      onMouseUp={handleMouseUp}
      onContextMenu={handleSystemContextMenu}
    >
      <rect
        x={-8}
        y={-8}
        height={16}
        width={16}
        fill="black"
        stroke="black"
        strokeWidth={1}
        rx={3}
        ry={3}
      />
      <g opacity={opacity}>
        <rect
          className={[
            "map-system-core",
            `map-threat-${threatStage}`,
            alerting ? "map-system-alert" : "",
            clearFlashing ? "map-system-clear-flash" : "",
          ]
            .filter(Boolean)
            .join(" ")}
          x={-8}
          y={-8}
          height={16}
          width={16}
          fill={systemFill}
          stroke={systemFill}
          strokeWidth={1.5}
          rx={3}
          ry={3}
          onMouseEnter={showSystemHoverCard}
          onMouseLeave={hideTimerTooltip}
        />
        <SystemBorderDisplay border={systemBorder} />
        <SystemHoverCard
          open={timerTooltipOpen}
          anchor={timerTooltipAnchor}
          systemName={system.name}
          characters={locatedCharacters}
          timerSignal={timerSignal}
          onMouseEnter={() => {
            if (hideTimerTooltipTimeoutRef.current) {
              window.clearTimeout(hideTimerTooltipTimeoutRef.current);
              hideTimerTooltipTimeoutRef.current = undefined;
            }
            setIsHoverCardHovered(true);
          }}
          onMouseLeave={() => {
            setIsHoverCardHovered(false);
            hideTimerTooltip();
          }}
        />
        {showSystemText && (
          <text
            className="map-system-name"
            transform="translate(0 19)"
            fill={textColor}
          >
            {system.name}
          </text>
        )}
      </g>
      {timerSignal && (
        <g transform="translate(6 -6)">
          {(() => {
            const dotSeverity = highestSignalSeverity(timerSignal);
            const dotColor = timerSignalColor(dotSeverity);
            const imminent = isTimerImminent(timerSignal.next_expires_at);
            return (
              <circle
                className={[
                  "map-system-timer-badge",
                  timerSeverityGlowClass(dotSeverity),
                  imminent ? "map-system-timer-badge-imminent" : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
                r={3.5}
                fill={dotColor}
                stroke={imminent ? dotColor : "none"}
                strokeWidth={imminent ? 1.1 : 0}
              />
            );
          })()}
        </g>
      )}
    </g>
  );
}

function isTimerImminent(expiresAt: string): boolean {
  const targetMs = Date.parse(expiresAt);
  if (Number.isNaN(targetMs)) return false;
  const remainingMs = targetMs - Date.now();
  return remainingMs > 0 && remainingMs <= 30 * 60 * 1000;
}

function timerSignalColor(severity: string): string {
  return timerSeverityDotColor(severity);
}

function timerSeverityGlowClass(severity: string): string {
  return `map-system-timer-badge-${normalizeTimerSeverity(severity)}`;
}

function highestSignalSeverity(signal: {
  severity: string;
  timers?: Array<{ severity: string }>;
}): string {
  let best = signal.severity;
  let bestRank = timerSeverityRank(best);
  for (const timer of signal.timers ?? []) {
    const rank = timerSeverityRank(timer.severity);
    if (rank > bestRank) {
      best = timer.severity;
      bestRank = rank;
    }
  }
  return best;
}
