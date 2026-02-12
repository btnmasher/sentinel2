import { useEffect, useMemo, useState } from "react";
import { useShallow } from "zustand/shallow";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";
import { useAdminSearchStore } from "../store/adminSearchStore";
import { useAdminActionsStore } from "../store/adminActionsStore";
import { buildSearchLabel, hasSearchMain } from "../utils/formatters";
import type { SearchResult } from "../types";

function MergeAccountModalBody() {
  const { close } = useModalBody();
  const user = useAdminStore((s) => s.selectedUser);
  const { query, results, loading, setQuery, clearSearch } =
    useAdminSearchStore(
      useShallow((s) => ({
        query: s.searches.merge.query,
        results: s.searches.merge.results,
        loading: s.searches.merge.loading,
        setQuery: s.setQuery,
        clearSearch: s.clear,
      })),
    );
  const mergeUser = useAdminActionsStore((s) => s.mergeUser);
  const [target, setTarget] = useState<SearchResult | null>(null);
  const [targetLabel, setTargetLabel] = useState("");

  useEffect(() => {
    setTarget(null);
    setTargetLabel("");
    clearSearch("merge");
  }, [clearSearch, user?.user_id]);

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

  const targetHasMain = hasSearchMain(target);
  if (!user) return null;

  return (
    <div className="text-sm text-slate-300 space-y-3">
      <div className="space-y-2">
        <label className="text-xs text-slate-400">Destination user</label>
        <input
          className="input input-xs input-bordered bg-base-300 w-full"
          placeholder="Search destination user"
          value={query}
          onChange={(e) => setQuery("merge", e.target.value)}
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
          onClick={() => void mergeUser(target)}
          disabled={!target?.user_id || !targetHasMain}
        >
          Merge account
        </button>
      </div>
    </div>
  );
}

export const AdminModalMerge = defineAdminModal({
  key: ADMIN_MODAL.Merge,
  useOpen: () => {
    const open = useAdminStore((s) => s.modals[ADMIN_MODAL.Merge]);
    const user = useAdminStore((s) => s.selectedUser);
    return open && Boolean(user);
  },
  build: () => ({
    title: "Merge Accounts",
    sizeClass: "max-w-lg",
    body: <MergeAccountModalBody />,
  }),
});

export default function MergeAccountModal() {
  useModal(AdminModalMerge);

  return null;
}
