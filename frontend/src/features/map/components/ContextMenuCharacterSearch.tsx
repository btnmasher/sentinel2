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

const parseJsonSafe = (raw: string): unknown => {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  try {
    return JSON.parse(trimmed);
  } catch {
    const firstObject = trimmed.match(/\{[\s\S]*\}/)?.[0];
    if (firstObject) {
      return JSON.parse(firstObject);
    }
    const firstArray = trimmed.match(/\[[\s\S]*\]/)?.[0];
    if (firstArray) {
      return JSON.parse(firstArray);
    }
    throw new SyntaxError("Unable to parse JSON response");
  }
};

const fetchJson = async (url: string, init?: RequestInit): Promise<unknown> => {
  const res = await fetch(url, init);
  const text = await res.text();
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${text.slice(0, 200)}`);
  }
  return parseJsonSafe(text);
};

const fetchJsonWithFallback = async (
  urls: string[],
  init?: RequestInit,
): Promise<unknown> => {
  let lastError: unknown;
  for (const url of urls) {
    try {
      return await fetchJson(url, init);
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError ?? new Error("No URL candidates succeeded");
};

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
        const encoded = encodeURIComponent(text);
        const searchData = (await fetchJsonWithFallback([
          `${ESI_BASE}/v2/search/?categories=character&search=${encoded}&strict=false`,
          `${ESI_BASE}/latest/search/?categories=character&search=${encoded}&strict=false`,
        ])) as { character?: number[] };
        if (
          !Array.isArray(searchData.character) ||
          searchData.character.length === 0
        ) {
          if (!cancelled) setCharacters([]);
          return;
        }
        const names = (await fetchJsonWithFallback(
          [
            `${ESI_BASE}/v2/universe/names/`,
            `${ESI_BASE}/latest/universe/names/`,
          ],
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(searchData.character),
          },
        )) as Character[];
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
