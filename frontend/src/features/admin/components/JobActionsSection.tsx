import { useMemo, useState } from "react";
import useConfirm from "@/app/hooks/useConfirm";
import Panel from "@/components/Panel";
import { useAdminMapDataStore } from "../store/adminMapDataStore";
import { ADMIN_MODAL } from "../store/adminStore";
import { useAdminModal } from "./AdminModals";

type ActionGroup = "map_data" | "characters" | "maintenance";

const GROUP_LABELS: Record<ActionGroup, string> = {
  map_data: "Map Data",
  characters: "Characters",
  maintenance: "Maintenance",
};

export default function JobActionsSection() {
  const requestConfirm = useConfirm();
  const { open: openAnnouncementModal } = useAdminModal(
    ADMIN_MODAL.Announcement,
  );
  const loadingLabel = useAdminMapDataStore((s) => s.loadingLabel);
  const runAction = useAdminMapDataStore((s) => s.runAction);
  const [group, setGroup] = useState<ActionGroup>("map_data");

  const actions = useMemo(() => {
    switch (group) {
      case "characters":
        return [
          {
            label: "Refresh all characters",
            path: "/admin/characters/refresh-all",
            confirm: "Start full character refresh?",
          },
        ];
      case "maintenance":
        return [
          {
            label: "Run cleanup job",
            path: "/admin/jobs/cleanup",
            confirm: "Start cleanup job now?",
          },
        ];
      default:
        return [
          {
            label: "Run full update",
            path: "/admin/map-data/run",
            confirm: "Start the full map data update?",
          },
          { label: "SDE import", path: "/admin/map-data/sde" },
          {
            label: "Build real positions",
            path: "/admin/map-data/real-positions",
          },
          {
            label: "Build eve2d positions",
            path: "/admin/map-data/eve2d-positions",
          },
          { label: "Dotlan import", path: "/admin/map-data/dotlan" },
          { label: "Import planets", path: "/admin/map-data/planets" },
          { label: "Import moons", path: "/admin/map-data/moons" },
          {
            label: "Build metro positions",
            path: "/admin/map-data/metro-positions",
          },
        ];
    }
  }, [group]);
  const panelActions = (
    <select
      className="select select-xs bg-base-300/70"
      value={group}
      onChange={(event) => setGroup(event.target.value as ActionGroup)}
    >
      {Object.entries(GROUP_LABELS).map(([key, label]) => (
        <option key={key} value={key}>
          {label}
        </option>
      ))}
    </select>
  );

  return (
    <Panel
      title="Job Actions"
      hint="Trigger background jobs and updates."
      actions={panelActions}
      bodyClassName="space-y-4"
    >
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <button
            key={action.path}
            className="btn btn-xs btn-info btn-outline"
            onClick={() => {
              if (action.confirm) {
                requestConfirm({
                  title: action.label,
                  body: action.confirm,
                  onConfirm: () => runAction(action.label, action.path),
                  confirmLabel: "Run",
                  cancelLabel: "Cancel",
                  tone: "default",
                });
                return;
              }
              void runAction(action.label, action.path);
            }}
            disabled={loadingLabel !== null}
          >
            {action.label}
          </button>
        ))}
        <button
          className="btn btn-xs btn-outline"
          onClick={openAnnouncementModal}
          disabled={loadingLabel !== null}
        >
          Publish announcement
        </button>
      </div>
      {loadingLabel && (
        <p className="text-xs text-slate-400">Running: {loadingLabel}…</p>
      )}
    </Panel>
  );
}
