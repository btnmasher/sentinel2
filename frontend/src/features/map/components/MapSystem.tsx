import { useEffect, useRef, useState } from "react";
import { useIntelStore } from "@/features/intel";
import { useMapStore } from "../store/mapStore";
import { colorForAge, colorToHex, transformComponent } from "../utils/mapUtils";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useOpenSystemContextMenu } from "../hooks/useOpenSystemContextMenu";

export default function MapSystem({
  systemId,
  scale,
}: {
  systemId: number;
  scale: number;
}) {
  const systems = useMapStore((s) => s.systems);
  const mapScale = useMapStore((s) => s.mapScale);
  const jumpranges = useMapStore((s) => s.jumpranges);
  const setJumpranges = useMapStore((s) => s.setJumpranges);
  const mapSettings = useSettingsStore((s) => s.settings.map);
  const intelSettings = useSettingsStore((s) => s.settings.intel);
  const characterLocations = useMapStore((s) => s.characterLocations);
  const characterInSpace = useMapStore((s) => s.characterInSpace);
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const characters = useMapStore((s) => s.characters);
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const routeWaypointsByCharacter = useMapStore(
    (s) => s.routeWaypointsByCharacter,
  );

  const logFilters = useIntelStore((s) => s.logFilters);
  const lastIntelSystems = useIntelStore((s) => s.lastIntelSystems);
  const openSystemContextMenu = useOpenSystemContextMenu();

  const [intelAgeSeconds, setIntelAgeSeconds] = useState<number | undefined>(
    undefined,
  );
  const timeoutRef = useRef<number | undefined>(undefined);

  const system = systems[systemId];

  useEffect(() => {
    if (timeoutRef.current) {
      window.clearTimeout(timeoutRef.current);
      timeoutRef.current = undefined;
    }
    const reportTime = lastIntelSystems[systemId];
    if (reportTime === undefined) {
      setIntelAgeSeconds(undefined);
      return;
    }

    const flashSeconds = intelSettings.flashEnabled
      ? Math.max(0, Math.floor(intelSettings.flashSeconds))
      : 0;
    const fadeSeconds = intelSettings.fadeEnabled
      ? Math.max(0, Math.floor(intelSettings.fadeSeconds))
      : 0;
    const activeSeconds = flashSeconds + fadeSeconds;
    const computeElapsed = () =>
      Math.max(0, Math.floor(Date.now() / 1000) - reportTime);

    const elapsed = computeElapsed();
    setIntelAgeSeconds(elapsed);
    if (activeSeconds <= 0 || elapsed >= activeSeconds) {
      setIntelAgeSeconds(undefined);
      return;
    }

    const tick = () => {
      const nextElapsed = computeElapsed();
      if (nextElapsed >= activeSeconds) {
        setIntelAgeSeconds(undefined);
        timeoutRef.current = undefined;
        return;
      }
      setIntelAgeSeconds(nextElapsed);
      const delay = nextElapsed < flashSeconds ? 1000 : 15000;
      timeoutRef.current = window.setTimeout(tick, delay);
    };

    timeoutRef.current = window.setTimeout(
      tick,
      elapsed < flashSeconds ? 1000 : 15000,
    );
    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
        timeoutRef.current = undefined;
      }
    };
  }, [
    intelSettings.fadeEnabled,
    intelSettings.fadeSeconds,
    intelSettings.flashEnabled,
    intelSettings.flashSeconds,
    lastIntelSystems,
    systemId,
  ]);

  if (!system) return null;

  const showSystem = mapSettings.alwaysShowSystems || mapScale > 0.4;
  const showSystemText = mapSettings.alwaysShowSystems || mapScale > 0.85;
  const opacity = jumpranges.enabled ? 1 : 0.65;
  const systemFill = colorForAge(
    intelAgeSeconds,
    intelSettings.flashEnabled ? intelSettings.flashSeconds : 0,
    intelSettings.fadeEnabled ? intelSettings.fadeSeconds : 0,
  );
  const textColor = logFilters.system.includes(systemId)
    ? colorToHex("green lighten-1")
    : "rgba(255, 255, 255, 0.8)";
  const alerting =
    intelAgeSeconds !== undefined &&
    intelSettings.flashEnabled &&
    intelAgeSeconds < Math.max(0, Math.floor(intelSettings.flashSeconds));
  const locatedCharacters = characters
    .filter((char) => visibleCharacterIds.includes(char.id))
    .filter((char) => characterLocations[char.id] === systemId)
    .map((char) => ({
      ...char,
      inSpace: characterInSpace[char.id] !== false,
    }));

  const routeWaypoints =
    lastRouteCharacter !== undefined
      ? (routeWaypointsByCharacter[lastRouteCharacter] ?? [])
      : [];
  const isRouteWaypoint = routeWaypoints.includes(systemId);
  const routeOrigin =
    lastRouteCharacter !== undefined
      ? characterLocations[lastRouteCharacter]
      : undefined;
  const isRouteOrigin = routeOrigin === systemId;

  const handleMouseUp = (event: React.MouseEvent<SVGGElement>) => {
    if (event.button !== 0) return;
    if (event.shiftKey) {
      useIntelStore.getState().toggleSystemFilter(systemId);
      return;
    }
    setJumpranges({ selectedSystem: systemId });
  };

  const handleContextMenu = (event: React.MouseEvent<SVGGElement>) => {
    openSystemContextMenu(event, systemId);
  };

  return (
    <g
      className={[
        "map-system",
        isRouteWaypoint ? "map-system-waypoint" : "",
        isRouteOrigin ? "map-system-route-origin" : "",
      ]
        .filter(Boolean)
        .join(" ")}
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
        {locatedCharacters.length > 0 && (
          <g transform="translate(0 -14)">
            {locatedCharacters.slice(0, 4).map((char, idx) => {
              const x = -6 + idx * 4;
              if (char.inSpace) {
                return (
                  <circle
                    key={char.id}
                    cx={x}
                    cy={0}
                    r={2.5}
                    fill="#38bdf8"
                    stroke="black"
                    strokeWidth={0.5}
                  >
                    <title>{char.name} (in space)</title>
                  </circle>
                );
              }
              return (
                <rect
                  key={char.id}
                  x={x - 2.5}
                  y={-2.5}
                  width={5}
                  height={5}
                  rx={1}
                  fill="#94a3b8"
                  stroke="black"
                  strokeWidth={0.5}
                >
                  <title>{char.name} (docked)</title>
                </rect>
              );
            })}
            {locatedCharacters.length > 4 && (
              <text x={10} y={2} fontSize={6} fill="white" opacity={0.9}>
                +{locatedCharacters.length - 4}
              </text>
            )}
          </g>
        )}
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
