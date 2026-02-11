import { EveType } from "@/types/eve";

type EveImageOptions = {
  size?: number;
};

const DEFAULT_SIZE: Record<EveType, number> = {
  [EveType.Alliance]: 32,
  [EveType.Corporation]: 32,
  [EveType.Character]: 64,
};

export function useEveImage(
  type: EveType,
  id?: number,
  options?: EveImageOptions,
) {
  if (!id) return "";
  const size = options?.size ?? DEFAULT_SIZE[type];
  switch (type) {
    case EveType.Alliance:
      return `https://images.evetech.net/alliances/${id}/logo?size=${size}`;
    case EveType.Corporation:
      return `https://images.evetech.net/corporations/${id}/logo?size=${size}`;
    case EveType.Character:
      return `https://images.evetech.net/characters/${id}/portrait?size=${size}`;
    default:
      return "";
  }
}

export function useAllianceLogo(allianceId?: number, size?: number) {
  return useEveImage(EveType.Alliance, allianceId, { size });
}

export function useCorporationLogo(corpId?: number, size?: number) {
  return useEveImage(EveType.Corporation, corpId, { size });
}

export function useCharacterPortrait(characterId?: number, size?: number) {
  return useEveImage(EveType.Character, characterId, { size });
}
