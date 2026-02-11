import { useEffect } from "react";
import { useMapStore } from "../store/mapStore";

export default function CharacterLocationRefresher() {
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const refreshCharacterLocations = useMapStore(
    (s) => s.refreshCharacterLocations,
  );

  useEffect(() => {
    if (visibleCharacterIds.length === 0) {
      return;
    }
    refreshCharacterLocations(visibleCharacterIds);
    const interval = window.setInterval(() => {
      refreshCharacterLocations(visibleCharacterIds);
    }, 30000);
    return () => window.clearInterval(interval);
  }, [refreshCharacterLocations, visibleCharacterIds]);

  return null;
}
