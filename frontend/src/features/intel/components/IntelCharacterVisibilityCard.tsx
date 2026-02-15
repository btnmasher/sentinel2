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
  dockedSystemName?: string;
  checked: boolean;
  onToggle: (checked: boolean) => void;
};

function CharacterVisibilityRow({
  id,
  name,
  inSpace,
  dockedSystemName,
  checked,
  onToggle,
}: CharacterVisibilityRowProps) {
  const portraitUrl = useCharacterPortrait(id, 32);

  return (
    <label className="flex items-center gap-2 text-xs">
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
      <span className="flex-1">{name}</span>
      <span
        className={
          inSpace === false
            ? "inline-block w-2.5 h-2.5 bg-slate-400 rounded-[3px]"
            : "inline-block w-2.5 h-2.5 bg-sky-400 rounded-full"
        }
      />
      {inSpace === false && (
        <>
          <span className="text-[10px] uppercase text-slate-500">docked</span>
          {dockedSystemName ? (
            <span className="text-[10px] text-slate-400 max-w-[120px] truncate">
              · {dockedSystemName}
            </span>
          ) : null}
        </>
      )}
    </label>
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
  const systems = useMapStore((s) => s.systems);
  const selectAllCharacters = useMapStore((s) => s.selectAllCharacters);
  const selectNoCharacters = useMapStore((s) => s.selectNoCharacters);
  const setVisibleCharacters = useMapStore((s) => s.setVisibleCharacters);

  const inSpaceCount = visibleCharacterIds.filter(
    (id) => characterInSpace[id] !== false,
  ).length;
  const dockedCount = visibleCharacterIds.filter(
    (id) => characterInSpace[id] === false,
  ).length;

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
                dockedSystemName={
                  characterInSpace[char.id] === false
                    ? systems[characterLocations[char.id]]?.name
                    : undefined
                }
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
