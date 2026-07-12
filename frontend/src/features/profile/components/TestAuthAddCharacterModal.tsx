import { useEffect, useState } from "react";
import useModal from "@/app/hooks/useModal";
import { api } from "@/config/api";
import { useMapStore } from "@/features/map";
import { useUIStore } from "@/app/store/uiStore";
import { getErrorMessage, getHttpData, getHttpStatus } from "@/utils/httpError";

type LinkableCharacter = {
  character_id: number;
  name: string;
  is_primary: boolean;
  has_valid_token: boolean;
};

type LinkableCharactersResponse = {
  characters?: LinkableCharacter[];
};

type TestAuthAddCharacterModalProps = {
  open: boolean;
  onClose: () => void;
  onLinked: () => Promise<void> | void;
};

export default function TestAuthAddCharacterModal({
  open,
  onClose,
  onLinked,
}: TestAuthAddCharacterModalProps) {
  const setToast = useUIStore((s) => s.setToast);
  const [characters, setCharacters] = useState<LinkableCharacter[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<number[]>([]);

  useEffect(() => {
    if (!open) {
      setCharacters([]);
      setSelected([]);
      setQuery("");
      setLoading(false);
      setSaving(false);
      return;
    }

    let mounted = true;
    setLoading(true);
    setCharacters([]);
    setSelected([]);
    setQuery("");

    api
      .get("/auth/characters/linkable")
      .then((res) => {
        if (!mounted) return;
        const payload = res.data as LinkableCharactersResponse;
        setCharacters(payload.characters || []);
      })
      .catch((error: unknown) => {
        if (!mounted) return;
        setToast({
          text: getErrorMessage(error, "Failed to load available characters"),
          color: "error",
          meta: {
            scope: "profile",
            operation: "load_linkable_characters",
            status: getHttpStatus(error),
            data: getHttpData(error),
          },
        });
      })
      .finally(() => {
        if (mounted) {
          setLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [open, setToast]);

  const toggleCharacter = (characterID: number) => {
    setSelected((current) =>
      current.includes(characterID)
        ? current.filter((id) => id !== characterID)
        : [...current, characterID],
    );
  };

  const filteredCharacters = characters.filter((character) => {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return (
      character.name.toLowerCase().includes(needle) ||
      String(character.character_id).includes(needle)
    );
  });

  const linkSelected = async () => {
    if (selected.length === 0 || saving) return;

    setSaving(true);
    try {
      await api.post("/auth/characters", {
        character_ids: selected,
      });
      useMapStore.getState().invalidateCharactersCache();
      await onLinked();
      onClose();
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to link character"),
        color: "error",
        meta: {
          scope: "profile",
          operation: "link_characters",
          status: getHttpStatus(error),
          data: getHttpData(error),
        },
      });
    } finally {
      setSaving(false);
    }
  };

  useModal({
    open,
    onDismiss: onClose,
    build: () => ({
      title: "Add Characters",
      body: (
        <>
          <p className="text-sm text-slate-400">
            Choose one or more unlinked characters from your TestAuth profile.
          </p>

          <label className="form-control w-full">
            <span className="label-text mb-1 text-xs uppercase tracking-wide text-slate-500">
              Search
            </span>
            <input
              className="input input-bordered w-full"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search by character name or ID"
            />
          </label>

          {loading ? (
            <p className="text-sm text-slate-400">
              Loading available characters…
            </p>
          ) : filteredCharacters.length === 0 ? (
            <p className="text-sm text-slate-400">
              No unlinked TestAuth characters are available.
            </p>
          ) : (
            <div className="max-h-[48vh] space-y-2 overflow-y-auto pr-1">
              {filteredCharacters.map((character) => {
                const checked = selected.includes(character.character_id);
                return (
                  <label
                    key={character.character_id}
                    className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-800 bg-base-300/60 px-3 py-2 transition hover:border-slate-600"
                  >
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-sm mt-1"
                      checked={checked}
                      onChange={() => toggleCharacter(character.character_id)}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="truncate font-medium">{character.name}</p>
                        {character.is_primary && (
                          <span className="badge badge-primary badge-sm">
                            Main
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-slate-400">
                        Character ID {character.character_id}
                      </p>
                    </div>
                  </label>
                );
              })}
            </div>
          )}
        </>
      ),
      actions: (
        <>
          <button className="btn btn-ghost" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button
            className="btn btn-primary"
            onClick={() => void linkSelected()}
            disabled={saving || loading || selected.length === 0}
          >
            Link selected
          </button>
        </>
      ),
      sizeClass: "max-w-2xl",
      closeDisabled: saving,
      dismissible: !saving,
    }),
  });

  return null;
}
