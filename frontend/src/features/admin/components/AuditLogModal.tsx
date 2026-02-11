import { useEffect } from "react";
import { useShallow } from "zustand/shallow";
import PaginationControls from "@/components/PaginationControls";
import SelectionDropdown from "@/components/SelectionDropdown";
import { useAdminAuditStore } from "../store/adminAuditStore";
import { useAdminStore } from "../store/adminStore";
import { formatDateTime } from "../utils/formatters";

export default function AuditLogModal() {
  const open = useAdminStore((s) => s.modals.audit);
  const userId = useAdminStore((s) => s.selectedUser?.user_id ?? null);
  const setModal = useAdminStore((s) => s.setModal);

  const {
    entries,
    loading,
    page,
    hasMore,
    action,
    actor,
    summary,
    setAction,
    setActor,
    setSummary,
    setPage,
    clear,
    fetchAudit,
  } = useAdminAuditStore(
    useShallow((s) => ({
      entries: s.entries,
      loading: s.loading,
      page: s.page,
      hasMore: s.hasMore,
      action: s.action,
      actor: s.actor,
      summary: s.summary,
      setAction: s.setAction,
      setActor: s.setActor,
      setSummary: s.setSummary,
      setPage: s.setPage,
      clear: s.clear,
      fetchAudit: s.fetchAudit,
    })),
  );

  const actionOptions = [
    { id: "all", label: "All actions" },
    { id: "user.", label: "User actions" },
    { id: "user.access", label: "Access changes" },
    { id: "user.sessions", label: "Session revokes" },
    { id: "user.uploaders", label: "Uploader revokes" },
    { id: "character.", label: "Character actions" },
    { id: "character.main", label: "Main changes" },
    { id: "character.move", label: "Character moves" },
    { id: "character.merge", label: "Account merges" },
    { id: "character.revoke", label: "Character revokes" },
    { id: "character.refresh", label: "Character refresh" },
  ];

  useEffect(() => {
    if (!open || !userId) {
      clear();
      return;
    }
    setPage(1);
    void fetchAudit(userId, 1);
  }, [clear, fetchAudit, open, setPage, userId]);

  if (!open || !userId) return null;

  return (
    <div className="modal modal-open">
      <div className="modal-box bg-base-200 border border-slate-700 max-w-2xl">
        <h3 className="font-display text-lg mb-3">Activity Log</h3>
        <div className="text-sm text-slate-300 space-y-3">
          <div className="grid gap-2 text-xs">
            <SelectionDropdown
              items={actionOptions}
              selected={[action]}
              onChange={(next) => setAction(next[0] ?? "all")}
              label="Action filter"
              buttonClassName="w-full"
            />
            <input
              className="input input-xs input-bordered bg-base-300"
              placeholder="Filter actor"
              value={actor}
              onChange={(e) => setActor(e.target.value)}
            />
            <input
              className="input input-xs input-bordered bg-base-300"
              placeholder="Filter summary"
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
            />
            <div className="flex gap-2">
              <button
                className="btn btn-xs btn-outline"
                onClick={() => {
                  setPage(1);
                  void fetchAudit(userId, 1, { action, actor, summary });
                }}
              >
                Apply
              </button>
              <button
                className="btn btn-xs btn-outline"
                onClick={() => {
                  setAction("all");
                  setActor("");
                  setSummary("");
                  setPage(1);
                  void fetchAudit(userId, 1, {
                    action: "all",
                    actor: "",
                    summary: "",
                  });
                }}
              >
                Clear
              </button>
            </div>
          </div>
          {loading ? (
            <p className="text-xs text-slate-400">Loading activity…</p>
          ) : entries.length === 0 ? (
            <p className="text-xs text-slate-400">No activity recorded.</p>
          ) : (
            <ul className="space-y-2 text-xs">
              {entries.map((entry) => (
                <li
                  key={entry.id}
                  className="border border-slate-800/70 rounded-lg px-3 py-2"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="font-semibold">{entry.action}</span>
                    <span className="text-slate-400">
                      {formatDateTime(entry.created)}
                    </span>
                  </div>
                  <p className="text-slate-300">{entry.summary}</p>
                  <div className="text-slate-500 mt-1">
                    {entry.actor_display_name || entry.actor_id}
                  </div>
                </li>
              ))}
            </ul>
          )}
          {(page > 1 || hasMore) && (
            <PaginationControls
              page={page}
              hasMore={hasMore}
              loading={loading}
              onPrev={() => {
                const nextPage = Math.max(1, page - 1);
                setPage(nextPage);
                void fetchAudit(userId, nextPage);
              }}
              onNext={() => {
                const nextPage = page + 1;
                setPage(nextPage);
                void fetchAudit(userId, nextPage);
              }}
            />
          )}
        </div>
        <button
          className="btn btn-outline btn-sm btn-square absolute right-3 top-3"
          onClick={() => setModal("audit", false)}
        >
          ✕
        </button>
      </div>
    </div>
  );
}
