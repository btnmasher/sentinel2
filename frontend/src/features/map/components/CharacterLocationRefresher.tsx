import { useEffect } from "react";
import { useMapStore } from "../store/mapStore";

export default function CharacterLocationRefresher() {
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const refreshCharacterLocations = useMapStore(
    (s) => s.refreshCharacterLocations,
  );

  useEffect(() => {
    const targets = new Set(visibleCharacterIds);
    if (lastRouteCharacter) {
      targets.add(lastRouteCharacter);
    }
    const pollTargets = Array.from(targets);
    if (pollTargets.length === 0) {
      return;
    }
    refreshCharacterLocations(pollTargets);
    const interval = window.setInterval(() => {
      refreshCharacterLocations(pollTargets);
    }, 30000);
    return () => window.clearInterval(interval);
  }, [lastRouteCharacter, refreshCharacterLocations, visibleCharacterIds]);

  return null;
}
