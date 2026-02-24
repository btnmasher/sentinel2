import { useEffect } from "react";
import { useShallow } from "zustand/shallow";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import CursorPagination from "@/components/CursorPagination";
import SelectionDropdown from "@/components/SelectionDropdown";
import { useAdminAuditStore } from "../store/adminAuditStore";
import {
  ADMIN_MODAL,
  defineAdminModal,
  useAdminStore,
} from "../store/adminStore";
import type { AuditEntry } from "../types";
import { formatDateTime } from "../utils/formatters";

function formatAuditTarget(entry: AuditEntry): string {
  if (entry.target_label?.trim()) return entry.target_label.trim();
  if (entry.target_character_name?.trim())
    return entry.target_character_name.trim();
  if (entry.target_user_name?.trim()) return entry.target_user_name.trim();
  if (entry.target_id?.trim()) return entry.target_id.trim();
  return "n/a";
}

function AuditLogModalBody() {
  const { close } = useModalBody();
  const userId = useAdminStore((s) => s.selectedUser?.user_id ?? null);
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
    { id: "user.auth.", label: "User auth actions" },
    { id: "user.access", label: "Access changes" },
    { id: "user.sessions", label: "Session revokes" },
    { id: "user.uploaders", label: "Uploader revokes" },
    { id: "character.", label: "Character actions" },
    { id: "character.main", label: "Main changes" },
    { id: "character.move", label: "Character moves" },
    { id: "character.merge", label: "Account merges" },
    { id: "character.revoke", label: "Character revokes" },
    { id: "character.refresh", label: "Character refresh" },
    { id: "job.", label: "Job actions" },
    { id: "admin.map_data.", label: "Map data actions" },
    { id: "announcement.", label: "Announcement actions" },
    { id: "staff.channel.", label: "Staff channel actions" },
    { id: "staff.jumpbridge.", label: "Staff jumpbridge actions" },
  ];

  useEffect(() => {
    if (!userId) {
      clear();
      return;
    }
    setPage(1);
    void fetchAudit(userId, 1);
    return () => clear();
  }, [clear, fetchAudit, setPage, userId]);

  if (!userId) return null;

  return (
    <div className="text-sm text-slate-300 space-y-3">
      <div className="grid gap-2 text-xs">
        <SelectionDropdown
          items={actionOptions}
          selected={[action]}
          onChange={(next) => setAction(next[0] ?? "all")}
          label="Action filter"
          buttonClassName="w-full"
        />
        <div className="grid gap-2 md:grid-cols-2">
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
        </div>
        <div className="flex justify-end gap-2">
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
              <div className="mt-1 grid gap-1 text-slate-500">
                <div>
                  <span className="text-slate-400">Actor:</span>{" "}
                  {entry.actor_display_name || entry.actor_id || "n/a"}
                </div>
                <div>
                  <span className="text-slate-400">Target:</span>{" "}
                  {entry.target_type || "n/a"} / {formatAuditTarget(entry)}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
      {(page > 1 || hasMore) && (
        <CursorPagination
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
      <div className="modal-action">
        <button className="btn btn-xs btn-outline" onClick={() => close()}>
          Close
        </button>
      </div>
    </div>
  );
}

export const AdminModalAudit = defineAdminModal({
  key: ADMIN_MODAL.Audit,
  useOpen: () => {
    const open = useAdminStore((s) => s.modals[ADMIN_MODAL.Audit]);
    const userId = useAdminStore((s) => s.selectedUser?.user_id ?? null);
    return open && Boolean(userId);
  },
  build: () => ({
    title: "Activity Log",
    sizeClass: "max-w-2xl",
    body: <AuditLogModalBody />,
  }),
});

export default function AuditLogModal() {
  useModal(AdminModalAudit);

  return null;
}
