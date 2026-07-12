import { useMapStore } from "@/features/map";
import { useCharacterPortrait } from "@/hooks/useEveImage";
import AccordionCard from "@/components/AccordionCard";

type IntelCharacterVisibilityCardProps = {
  open: boolean;
  onToggle: () => void;
};

type CharacterVisibilityRowProps = {
  id: number;
  name: string;
  inSpace?: boolean;
  locationSystemId?: number;
  locationSystemName?: string;
  locationRegionId?: number;
  regionLoaded?: boolean;
  checked: boolean;
  onToggle: (checked: boolean) => void;
  onFocusLocation?: (systemId: number, regionId?: number) => void;
};

function CharacterVisibilityRow({
  id,
  name,
  inSpace,
  locationSystemId,
  locationSystemName,
  locationRegionId,
  regionLoaded,
  checked,
  onToggle,
  onFocusLocation,
}: CharacterVisibilityRowProps) {
  const portraitUrl = useCharacterPortrait(id, 32);
  const canFocus =
    Number.isFinite(locationSystemId) && Number(locationSystemId) > 0;
  const locationStateLabel = inSpace === false ? "docked" : "in space";
  const systemLabel =
    locationSystemName?.trim() ||
    (Number.isFinite(locationSystemId) && Number(locationSystemId) > 0
      ? `System ${locationSystemId}`
      : "");

  return (
    <div className="flex items-center gap-2 text-xs">
      <input
        type="checkbox"
        className="checkbox checkbox-xs rounded-[3px]"
        checked={checked}
        onChange={(event) => onToggle(event.target.checked)}
      />
      <img
        src={portraitUrl}
        alt={name}
        className="h-4 w-4 rounded-full"
        loading="lazy"
      />
      {canFocus && onFocusLocation ? (
        <button
          type="button"
          className="flex-1 truncate text-left text-sky-300 underline-offset-2 hover:text-sky-200 hover:underline"
          onClick={() =>
            onFocusLocation(Number(locationSystemId), locationRegionId)
          }
          title={
            regionLoaded
              ? "Focus this character's location on the map"
              : "Load the region and focus this character's location"
          }
        >
          {name}
        </button>
      ) : (
        <span className="flex-1">{name}</span>
      )}
      <span
        className={
          inSpace === false
            ? "inline-block w-2.5 h-2.5 bg-slate-400 rounded-[3px]"
            : "inline-block w-2.5 h-2.5 bg-sky-400 rounded-full"
        }
      />
      {systemLabel && (
        <div className="flex flex-wrap items-center gap-1 text-[10px] uppercase text-slate-500">
          <span className="text-[10px] uppercase text-slate-500">
            {locationStateLabel}
          </span>
          {onFocusLocation && canFocus ? (
            <button
              type="button"
              className={[
                "max-w-[160px] truncate text-left underline-offset-2 hover:underline",
                regionLoaded ? "text-slate-400 hover:text-sky-300" : "text-amber-300 hover:text-amber-200",
              ].join(" ")}
              onClick={() =>
                onFocusLocation(Number(locationSystemId), locationRegionId)
              }
              title={
                regionLoaded
                  ? "Focus this character's location on the map"
                  : "Load the region and focus this character's location"
              }
            >
              {systemLabel}
            </button>
          ) : (
            <span className="text-[10px] text-slate-400 max-w-[120px] truncate">
              {systemLabel}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export default function IntelCharacterVisibilityCard({
  open,
  onToggle,
}: IntelCharacterVisibilityCardProps) {
  const characters = useMapStore((s) => s.characters);
  const visibleCharacterIds = useMapStore((s) => s.visibleCharacterIds);
  const characterInSpace = useMapStore((s) => s.characterInSpace);
  const characterLocations = useMapStore((s) => s.characterLocations);
  const characterLocationSystemNames = useMapStore(
    (s) => s.characterLocationSystemNames,
  );
  const characterLocationRegions = useMapStore(
    (s) => s.characterLocationRegions,
  );
  const mapRegions = useMapStore((s) => s.mapRegions);
  const systems = useMapStore((s) => s.systems);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const selectAllCharacters = useMapStore((s) => s.selectAllCharacters);
  const selectNoCharacters = useMapStore((s) => s.selectNoCharacters);
  const setVisibleCharacters = useMapStore((s) => s.setVisibleCharacters);

  const inSpaceCount = visibleCharacterIds.filter(
    (id) => characterInSpace[id] !== false,
  ).length;
  const dockedCount = visibleCharacterIds.filter(
    (id) => characterInSpace[id] === false,
  ).length;

  const focusLocation = (systemId: number, regionId?: number) => {
    if (regionId && !mapRegions.includes(String(regionId))) {
      updateMapConfig({ mapRegions: [...mapRegions, String(regionId)] });
    }
    setSystemSearch(systemId);
  };

  const selectMainOnly = () => {
    const main = characters.find((char) => char.is_main);
    if (main) {
      setVisibleCharacters([main.id]);
      return;
    }
    if (characters.length > 0) {
      setVisibleCharacters([characters[0].id]);
    }
  };

  return (
    <AccordionCard
      title="Character Visibility"
      subtitle="Which characters show on the map"
      open={open}
      onToggle={onToggle}
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between text-xs">
          <span>Visibility</span>
          <div className="flex gap-2">
            <button
              className="btn btn-xs btn-outline"
              onClick={selectAllCharacters}
            >
              All
            </button>
            <button className="btn btn-xs btn-outline" onClick={selectMainOnly}>
              Main only
            </button>
            <button
              className="btn btn-xs btn-outline"
              onClick={selectNoCharacters}
            >
              None
            </button>
          </div>
        </div>

        {characters.length === 0 ? (
          <div className="text-xs text-slate-500">No characters loaded.</div>
        ) : (
          <div className="space-y-2 max-h-40 overflow-auto">
            {characters.map((char) => (
              <CharacterVisibilityRow
                key={char.id}
                id={char.id}
                name={char.name}
                inSpace={characterInSpace[char.id]}
                locationSystemId={characterLocations[char.id]}
                locationSystemName={
                  characterInSpace[char.id] === false
                    ? characterLocationSystemNames[char.id] ??
                      systems[characterLocations[char.id]]?.name
                    : undefined
                }
                locationRegionId={characterLocationRegions[char.id]}
                regionLoaded={
                  characterLocationRegions[char.id]
                    ? mapRegions.includes(String(characterLocationRegions[char.id]))
                    : false
                }
                onFocusLocation={focusLocation}
                checked={visibleCharacterIds.includes(char.id)}
                onToggle={(checked) => {
                  if (checked) {
                    setVisibleCharacters([...visibleCharacterIds, char.id]);
                  } else {
                    setVisibleCharacters(
                      visibleCharacterIds.filter((id) => id !== char.id),
                    );
                  }
                }}
              />
            ))}
          </div>
        )}

        <div className="text-[10px] text-slate-500 pt-2 border-t border-slate-800">
          Showing {inSpaceCount} in space
          {dockedCount > 0 ? ` · ${dockedCount} docked` : ""}
        </div>
      </div>
    </AccordionCard>
  );
}
