import { useEffect, useState } from "react";
import { ESI_BASE } from "@/config/esi";
import { useUIStore } from "@/app/store/uiStore";
import { toErrorMeta } from "@/utils/httpError";
import {
  ContextMenuItem,
  ContextMenuList,
  ContextMenuTitle,
} from "./ContextMenuUI";

type Character = { id: number; name: string };

export default function ContextMenuCharacterSearch({ text }: { text: string }) {
  const [characters, setCharacters] = useState<Character[] | undefined>(
    undefined,
  );
  const setMenu = useUIStore((s) => s.setContextMenu);
  const menu = useUIStore((s) => s.contextMenu);
  const setToast = useUIStore((s) => s.setToast);

  useEffect(() => {
    let cancelled = false;
    const search = async () => {
      setCharacters(undefined);
      try {
        const searchRes = await fetch(
          `${ESI_BASE}/v2/search/?categories=character&search=${encodeURIComponent(text)}&strict=false`,
        );
        const searchData = await searchRes.json();
        if (!searchData.character) {
          if (!cancelled) setCharacters([]);
          return;
        }
        const namesRes = await fetch(`${ESI_BASE}/v2/universe/names/`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(searchData.character),
        });
        const names = await namesRes.json();
        if (!cancelled) setCharacters(names as Character[]);
      } catch (error: unknown) {
        setToast({
          text: "Error searching for characters",
          color: "error",
          meta: {
            scope: "context-menu-character-search",
            operation: "esi_character_search",
            query: text,
            error: toErrorMeta(error),
          },
        });
      }
    };
    if (text) {
      search();
    }
    return () => {
      cancelled = true;
    };
  }, [setToast, text]);

  return (
    <ContextMenuList>
      <ContextMenuTitle>Pilot Matches</ContextMenuTitle>
      {!characters && <ContextMenuItem muted>Searching...</ContextMenuItem>}
      {characters && characters.length === 0 && (
        <ContextMenuItem muted>No characters found</ContextMenuItem>
      )}
      {characters &&
        characters.map((character) => (
          <ContextMenuItem
            key={character.id}
            onClick={() =>
              setMenu(
                menu
                  ? {
                      ...menu,
                      type: "character",
                      character: character.name,
                      characterId: character.id,
                    }
                  : null,
              )
            }
          >
            {character.name}
          </ContextMenuItem>
        ))}
    </ContextMenuList>
  );
}
