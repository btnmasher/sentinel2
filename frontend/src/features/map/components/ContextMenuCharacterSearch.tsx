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
type UniverseIDsResponse = {
  characters?: Character[];
};

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

const buildNameCandidates = (source: string): string[] => {
  const trimmed = source.trim();
  if (!trimmed) return [];
  const normalized = trimmed.replace(/\s+/g, " ");
  const unwrapped = normalized.replace(/^[^\w'-]+|[^\w'-]+$/g, "");
  const candidates = [trimmed, normalized, unwrapped]
    .map((value) => value.trim())
    .filter((value) => value.length > 0);
  return Array.from(new Set(candidates));
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
        const names = buildNameCandidates(text);
        if (names.length === 0) {
          if (!cancelled) setCharacters([]);
          return;
        }
        const resolved = (await fetchJson(
          `${ESI_BASE}/v3/universe/ids/?datasource=tranquility`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(names),
          },
        )) as UniverseIDsResponse;
        const matches = Array.isArray(resolved.characters)
          ? resolved.characters
          : [];
        if (matches.length === 0) {
          if (!cancelled) setCharacters([]);
          return;
        }
        if (!cancelled) setCharacters(matches);
      } catch (error: unknown) {
        if (!cancelled) setCharacters([]);
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
      {!characters && (
        <ContextMenuItem muted>
          Searching
          <span className="context-menu-loading-dots" aria-hidden="true">
            <span>.</span>
            <span>.</span>
            <span>.</span>
          </span>
        </ContextMenuItem>
      )}
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
