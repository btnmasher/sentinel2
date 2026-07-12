import { useEffect, useMemo, useState } from "react";
import { useShallow } from "zustand/shallow";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import SelectionDropdown from "@/components/SelectionDropdown";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAdminSearchStore } from "../store/adminSearchStore";
import { useAdminActionsStore } from "../store/adminActionsStore";
import { buildSearchLabel, hasSearchMain } from "../utils/formatters";
import type { SearchResult } from "../types";

function MoveCharacterModalBody() {
  const { close } = useModalBody();
  const user = useAdminStore((s) => s.selectedUser);
  const { query, results, loading, setQuery, clearSearch } =
    useAdminSearchStore(
      useShallow((s) => ({
        query: s.searches.move.query,
        results: s.searches.move.results,
        loading: s.searches.move.loading,
        setQuery: s.setQuery,
        clearSearch: s.clear,
      })),
    );
  const moveCharacter = useAdminActionsStore((s) => s.moveCharacter);
  const [target, setTarget] = useState<SearchResult | null>(null);
  const [targetLabel, setTargetLabel] = useState("");
  const [characterId, setCharacterId] = useState("");

  useEffect(() => {
    if (!user) return;
    const main = user.characters.find((character) => character.is_main);
    setCharacterId(main?.id || user.characters[0]?.id || "");
    setTarget(null);
    setTargetLabel("");
    clearSearch("move");
  }, [clearSearch, user]);

  const filteredResults = useMemo(
    () => results.filter((result) => result.user_id !== user?.user_id),
    [results, user?.user_id],
  );

  const formattedResults = useMemo(
    () =>
      filteredResults.map((result) => ({
        result,
        label: buildSearchLabel(result),
      })),
    [filteredResults],
  );

  const characterOptions = useMemo(
    () =>
      user?.characters.map((character) => ({
        id: character.id,
        label: `${character.name}${character.is_main ? " (main)" : ""}`,
      })) ?? [],
    [user],
  );

  const targetHasMain = hasSearchMain(target);
  if (!user) return null;

  return (
    <div className="text-sm text-slate-300 space-y-3">
      <div className="space-y-2">
        <label className="text-xs text-slate-400">Character</label>
        <SelectionDropdown
          items={characterOptions}
          selected={characterId ? [characterId] : []}
          onChange={(next) => setCharacterId(next[0] ?? "")}
          label="Character"
          searchable
          buttonClassName="w-full"
        />
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-400">Destination user</label>
        <input
          className="input input-xs input-bordered bg-base-300 w-full"
          placeholder="Search destination user (name or ID)"
          value={query}
          onChange={(e) => setQuery("move", e.target.value)}
        />
        {loading && <p className="text-xs text-slate-400">Searching…</p>}
        <ul className="space-y-2 text-xs max-h-32 overflow-auto">
          {formattedResults.map(({ result, label }) => (
            <li key={result.character_record_id}>
              <button
                className="text-left w-full hover:text-primary transition-colors"
                onClick={() => {
                  setTarget(result);
                  setTargetLabel(label);
                }}
              >
                {label}
              </button>
            </li>
          ))}
        </ul>
        {target?.user_id && (
          <p className="text-xs text-slate-400">
            Destination: {targetLabel || target.user_id}
          </p>
        )}
        {target?.user_id && (
          <p className="text-xs text-slate-400">
            Target main:{" "}
            {target.main_name || (target.is_main ? target.name : "—")}
          </p>
        )}
        {target?.user_id && !targetHasMain && (
          <p className="text-xs text-amber-300">
            Warning: Target account is missing a main character.
          </p>
        )}
      </div>
      <div className="modal-action">
        <button className="btn btn-xs btn-outline" onClick={() => close()}>
          Cancel
        </button>
        <button
          className="btn btn-xs btn-outline"
          onClick={() => void moveCharacter(characterId, target)}
          disabled={!target?.user_id || !targetHasMain}
        >
          Move character
        </button>
      </div>
    </div>
  );
}

export const AdminModalMove = defineAdminModal({
  key: ADMIN_MODAL.Move,
  useOpen: () => {
    const open = useAdminStore((s) => s.modals[ADMIN_MODAL.Move]);
    const user = useAdminStore((s) => s.selectedUser);
    const standaloneAuth = useAppConfigStore((s) => s.standaloneAuth);
    return open && Boolean(user) && standaloneAuth;
  },
  build: () => ({
    title: "Move Character",
    sizeClass: "max-w-lg",
    body: <MoveCharacterModalBody />,
  }),
});

export default function MoveCharacterModal() {
  useModal(AdminModalMove);

  return null;
}
