import { useIntelStore } from "@/features/intel";
import { useMapStore } from "../store/mapStore";
import { colorToHex, transformComponent } from "../utils/mapUtils";
import { useSystemRouteState } from "../hooks/useSystemRouteState";
import { useSystemThreatState } from "../hooks/useSystemThreatState";
import { useSystemVisibility } from "../hooks/useSystemVisibility";
import { useSystemCharacters } from "../hooks/useSystemCharacters";
import { useSystemInteractions } from "../hooks/useSystemInteractions";
import { useSystemBorderState } from "../hooks/useSystemBorderState";
import SystemBorderDisplay from "./SystemBorderDisplay";
import SystemCharacterTooltip from "./SystemCharacterTooltip";

export default function MapSystem({
  systemId,
  scale,
}: {
  systemId: number;
  scale: number;
}) {
  const systems = useMapStore((s) => s.systems);
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
  const { systemFill, alerting } = useSystemThreatState(systemId);

  const system = systems[systemId];

  if (!system) return null;

  const opacity = jumpranges.enabled ? 1 : 0.65;
  const textColor = logFilters.system.includes(systemId)
    ? colorToHex("green lighten-1")
    : "rgba(255, 255, 255, 0.8)";

  return (
    <g
      className="map-system"
      transform={transformComponent(system.position, scale)}
      style={{ display: showSystem ? "block" : "none" }}
      onMouseUp={handleMouseUp}
      onContextMenu={handleContextMenu}
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
          className={["map-system-core", alerting ? "map-system-alert" : ""]
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
        />
        <SystemBorderDisplay border={systemBorder} />
        <SystemCharacterTooltip characters={locatedCharacters} />
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
    </g>
  );
}
