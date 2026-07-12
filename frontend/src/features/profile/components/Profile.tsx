import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/config/api";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useUIStore } from "@/app/store/uiStore";
import { useMapStore } from "@/features/map";
import CharacterList, { type Character } from "@/components/CharacterList";
import Panel from "@/components/Panel";
import { getErrorMessage, getHttpData, getHttpStatus } from "@/utils/httpError";
import AddCharacterModal from "./TestAuthAddCharacterModal";

export default function Profile() {
  const setToast = useUIStore((s) => s.setToast);
  const authBackend = useAppConfigStore((s) => s.authBackend);
  const forceLogout = useAuthStore((s) => s.forceLogout);
  const [characters, setCharacters] = useState<Character[]>([]);
  const [loading, setLoading] = useState(true);
  const [addCharacterOpen, setAddCharacterOpen] = useState(false);
  const requestIdRef = useRef(0);

  const loadProfile = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);

    try {
      const res = await api.get("/auth/profile");
      if (requestIdRef.current !== requestId) return;
      setCharacters(res.data.characters || []);
    } catch (error: unknown) {
      if (requestIdRef.current !== requestId) return;
      setToast({
        text: getErrorMessage(error, "Failed to load profile"),
        color: "error",
        meta: {
          scope: "profile",
          operation: "load_profile",
          status: getHttpStatus(error),
          data: getHttpData(error),
        },
      });
    } finally {
      if (requestIdRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [setToast]);

  useEffect(() => {
    void loadProfile();
    return () => {
      requestIdRef.current += 1;
    };
  }, [loadProfile]);

  const removeCharacter = async (character: Character) => {
    if (!character.record_id) {
      setToast({
        text: "Missing character record id.",
        color: "error",
      });
      return;
    }

    try {
      const res = await api.delete(`/auth/characters/${character.record_id}`);
      setCharacters((current) =>
        current.filter((item) => item.record_id !== character.record_id),
      );
      useMapStore.getState().invalidateCharactersCache();

      if (res.data?.deleted_user) {
        forceLogout("Character removed.");
      }
    } catch (error: unknown) {
      setToast({
        text: getErrorMessage(error, "Failed to remove character"),
        color: "error",
        meta: {
          scope: "profile",
          operation: "remove_character",
          status: getHttpStatus(error),
          data: getHttpData(error),
        },
      });
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
      <Panel
        title="Linked Characters"
        hint="Manage the characters tied to your Sentinel account."
      >
        {loading ? (
          <p className="text-sm text-slate-400">Loading characters…</p>
        ) : (
          <CharacterList
            characters={characters}
            onRemove={removeCharacter}
            disableRemove={(character) =>
              authBackend === "testauth"
                ? character.is_main
                : character.is_main && characters.length > 1
            }
          />
        )}
      </Panel>

      <Panel
        title="Add Character"
        hint="Link additional characters from your TestAuth profile."
      >
        {authBackend === "testauth" ? (
          <div className="space-y-3">
            <p className="text-sm text-slate-400">
              Open the picker to choose one or more unlinked characters from
              your TestAuth profile.
            </p>
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={() => setAddCharacterOpen(true)}
            >
              Add character
            </button>
          </div>
        ) : (
          <p className="text-sm text-slate-400">
            Character linking is managed by the active auth backend.
          </p>
        )}
      </Panel>

      <AddCharacterModal
        open={addCharacterOpen}
        onClose={() => setAddCharacterOpen(false)}
        onLinked={loadProfile}
      />
    </div>
  );
}
