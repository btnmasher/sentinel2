import { useMemo } from "react";
import type { SystemCharacterBadge } from "./useSystemCharacters";

export type SystemBorderState = {
  visible: boolean;
  color?: string;
  pulse: boolean;
};

type SystemBorderInput = {
  isRouteOrigin: boolean;
  isRouteWaypoint: boolean;
  characters: SystemCharacterBadge[];
};

const BORDER_COLORS = {
  routeOrigin: "#22c55e",
  routeWaypoint: "#22c55e",
  undocked: "#38bdf8",
  docked: "#94a3b8",
} as const;

export function useSystemBorderState({
  isRouteOrigin,
  isRouteWaypoint,
  characters,
}: SystemBorderInput): SystemBorderState {
  return useMemo(() => {
    const hasCharacters = characters.length > 0;
    const hasUndocked = characters.some((character) => character.inSpace);
    const color = isRouteOrigin
      ? BORDER_COLORS.routeOrigin
      : hasUndocked
        ? BORDER_COLORS.undocked
        : hasCharacters
          ? BORDER_COLORS.docked
          : isRouteWaypoint
            ? BORDER_COLORS.routeWaypoint
            : undefined;

    return {
      visible: Boolean(color),
      color,
      // Pulse color priority: route head > undocked > docked.
      pulse: isRouteOrigin || hasCharacters,
    };
  }, [characters, isRouteOrigin, isRouteWaypoint]);
}
