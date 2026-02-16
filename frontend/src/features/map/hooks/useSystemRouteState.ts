import { useMemo } from "react";
import { useMapStore } from "../store/mapStore";
import { useIsCharacterVisible } from "./useCharacterVisibility";

export function useSystemRouteState(systemId: number) {
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const routeWaypointsByCharacter = useMapStore(
    (s) => s.routeWaypointsByCharacter,
  );
  const characterLocations = useMapStore((s) => s.characterLocations);
  const route = useMapStore((s) => s.route);
  const isRouteCharacterVisible = useIsCharacterVisible(lastRouteCharacter);

  return useMemo(() => {
    const hasActiveRoute =
      isRouteCharacterVisible &&
      lastRouteCharacter !== undefined &&
      Array.isArray(route) &&
      route.length > 0;
    const routeWaypoints =
      hasActiveRoute && lastRouteCharacter !== undefined
        ? (routeWaypointsByCharacter[lastRouteCharacter] ?? [])
        : [];
    const routeOriginFromLocation =
      hasActiveRoute && lastRouteCharacter !== undefined
        ? characterLocations[lastRouteCharacter]
        : undefined;
    const routeOrigin =
      typeof routeOriginFromLocation === "number"
        ? routeOriginFromLocation
        : hasActiveRoute && lastRouteCharacter !== undefined
          ? route[0]
          : undefined;

    return {
      isRouteWaypoint: routeWaypoints.includes(systemId),
      isRouteOrigin: routeOrigin === systemId,
    };
  }, [
    characterLocations,
    lastRouteCharacter,
    route,
    routeWaypointsByCharacter,
    systemId,
    isRouteCharacterVisible,
  ]);
}
