import { useMemo } from "react";
import { useMapStore } from "@/features/map";

export default function NavigationFavoritesCard() {
  const characters = useMapStore((s) => s.characters);
  const favoriteCharacters = useMapStore((s) => s.favoriteCharacters);
  const setFavoriteCharacters = useMapStore((s) => s.setFavoriteCharacters);

  const favoriteSet = useMemo(
    () => new Set(favoriteCharacters),
    [favoriteCharacters],
  );

  return (
    <div className="card bg-base-200/70 border border-slate-800">
      <div className="card-body">
        <h3 className="font-display text-lg">Character Favorites</h3>
        <div className="space-y-2 text-xs text-slate-300">
          {characters.map((char) => (
            <label key={char.id} className="flex items-center justify-between">
              <span>{char.name}</span>
              <input
                className="checkbox checkbox-xs"
                type="checkbox"
                checked={favoriteSet.has(char.id)}
                onChange={() => {
                  const next = favoriteSet.has(char.id)
                    ? favoriteCharacters.filter((id) => id !== char.id)
                    : [...favoriteCharacters, char.id];
                  setFavoriteCharacters(next);
                }}
              />
            </label>
          ))}
        </div>
      </div>
    </div>
  );
}
