import { useMemo } from "react";
import { useMapStore } from "../store/mapStore";
import { useVisibleCharacterIdSet } from "./useCharacterVisibility";

export type SystemCharacterBadge = {
  id: number;
  name: string;
  inSpace: boolean;
};

export function useSystemCharacters(systemId: number) {
  const characterLocations = useMapStore((s) => s.characterLocations);
  const characterInSpace = useMapStore((s) => s.characterInSpace);
  const characters = useMapStore((s) => s.characters);
  const visibleCharacterIdSet = useVisibleCharacterIdSet();

  return useMemo<SystemCharacterBadge[]>(
    () =>
      characters
        .filter((char) => visibleCharacterIdSet.has(char.id))
        .filter((char) => characterLocations[char.id] === systemId)
        .map((char) => ({
          id: char.id,
          name: char.name,
          inSpace: characterInSpace[char.id] !== false,
        })),
    [
      characterInSpace,
      characterLocations,
      characters,
      systemId,
      visibleCharacterIdSet,
    ],
  );
}
