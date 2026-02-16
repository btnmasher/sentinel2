import { useMemo } from "react";
import { useMapStore } from "../store/mapStore";

export function useVisibleCharacterIds() {
  return useMapStore((s) => s.visibleCharacterIds);
}

export function useVisibleCharacterIdSet() {
  const visibleCharacterIds = useVisibleCharacterIds();
  return useMemo(() => new Set(visibleCharacterIds), [visibleCharacterIds]);
}

export function useIsCharacterVisible(characterId?: number) {
  const visibleCharacterIdSet = useVisibleCharacterIdSet();
  return useMemo(
    () =>
      typeof characterId === "number" && visibleCharacterIdSet.has(characterId),
    [characterId, visibleCharacterIdSet],
  );
}
