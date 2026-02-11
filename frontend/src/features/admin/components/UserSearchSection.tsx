import { useMemo } from "react";
import { useShallow } from "zustand/shallow";
import { useUIStore } from "@/app/store/uiStore";
import { useAdminStore } from "../store/adminStore";
import { useAdminSearchStore } from "../store/adminSearchStore";
import { buildSearchLabel } from "../utils/formatters";

export default function UserSearchSection() {
  const setToast = useUIStore((s) => s.setToast);
  const loadUser = useAdminStore((s) => s.loadUser);
  const { query, results, loading, setQuery } = useAdminSearchStore(
    useShallow((s) => ({
      query: s.searches.user.query,
      results: s.searches.user.results,
      loading: s.searches.user.loading,
      setQuery: s.setQuery,
    })),
  );

  const formattedResults = useMemo(
    () =>
      results.map((result) => ({
        result,
        label: buildSearchLabel(result),
      })),
    [results],
  );

  const handleSelectUser = async (userId: string) => {
    try {
      await loadUser(userId);
    } catch {
      setToast({ text: "Failed to load user", color: "error" });
    }
  };

  return (
    <section className="card bg-base-200/70 border border-slate-800 h-full min-h-0">
      <div className="card-body flex flex-col gap-2 h-full min-h-0">
        <h2 className="font-display text-2xl">User Search</h2>
        <input
          className="input input-sm input-bordered bg-base-300"
          placeholder="Search by character name"
          value={query}
          onChange={(e) => setQuery("user", e.target.value)}
        />
        {loading && <p className="text-xs text-slate-400">Searching…</p>}
        {!loading &&
          query.trim().length >= 2 &&
          formattedResults.length === 0 && (
            <p className="text-xs text-slate-400">No results.</p>
          )}
        <ul className="space-y-2 text-sm flex-1 min-h-0 overflow-auto">
          {formattedResults.map(({ result, label }) => (
            <li key={result.character_record_id}>
              <button
                className="w-full text-left rounded-lg border border-slate-800 bg-base-300/40 px-3 py-2 transition hover:bg-base-300/70"
                onClick={() => void handleSelectUser(result.user_id)}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-slate-100">{label}</span>
                  <span
                    className={`badge badge-xs ${result.is_main ? "badge-primary" : "badge-ghost"}`}
                  >
                    {result.is_main ? "Main" : "Alt"}
                  </span>
                </div>
                {result.main_name && !result.is_main && (
                  <div className="text-[11px] text-slate-400">
                    Main: {result.main_name}
                  </div>
                )}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
