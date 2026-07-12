import { useEffect, useMemo, useState } from "react";
import useConfirm from "@/app/hooks/useConfirm";
import Panel from "@/components/Panel";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAdminMapDataStore } from "../store/adminMapDataStore";
import { ADMIN_MODAL } from "../store/adminStore";
import { useAdminModal } from "./AdminModals";

type ActionGroup = "map_data" | "characters" | "timer_data" | "maintenance";
type ActionGroupWithSite = ActionGroup | "site_settings";
type AdminAction = {
  label: string;
  path?: string;
  confirm?: string;
  onClick?: () => void;
};

const GROUP_LABELS: Record<ActionGroupWithSite, string> = {
  map_data: "Map Data",
  characters: "Characters",
  timer_data: "Timer Data",
  maintenance: "Maintenance",
  site_settings: "Site Settings",
};

export default function JobActionsSection() {
  const requestConfirm = useConfirm();
  const { open: openAnnouncementModal } = useAdminModal(
    ADMIN_MODAL.Announcement,
  );
  const { open: openAllowedOrganizationsModal } = useAdminModal(
    ADMIN_MODAL.AllowedOrganizations,
  );
  const loadingLabel = useAdminMapDataStore((s) => s.loadingLabel);
  const runAction = useAdminMapDataStore((s) => s.runAction);
  const standaloneAuth = useAppConfigStore((s) => s.standaloneAuth);
  const timersEnabled = useAppConfigStore((s) => s.timersEnabled);
  const timerSource = useAppConfigStore((s) => s.timerSource);
  const standaloneTimers = timersEnabled && timerSource === "standalone";
  const [group, setGroup] = useState<ActionGroupWithSite>("map_data");
  const availableGroups = useMemo<[ActionGroupWithSite, string][]>(() => {
    return (
      Object.entries(GROUP_LABELS) as [ActionGroupWithSite, string][]
    ).filter(([key]) => standaloneTimers || key !== "timer_data");
  }, [standaloneTimers]);

  const actions = useMemo<AdminAction[]>(() => {
    switch (group) {
      case "characters":
        if (!standaloneAuth) {
          return [];
        }
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
      case "timer_data":
        if (!standaloneTimers) {
          return [];
        }
        return [
          {
            label: "Sync sovereignty campaigns",
            path: "/admin/jobs/timers/sovereignty-campaign-sync",
            confirm: "Run sovereignty campaign sync now?",
          },
          {
            label: "Sync structure notifications",
            path: "/admin/jobs/timers/structure-notifications-sync",
            confirm: "Run structure notifications sync now?",
          },
        ];
      case "site_settings":
        return [
          {
            label: "Allowed organizations",
            onClick: () => openAllowedOrganizationsModal(),
          },
          {
            label: "Publish announcement",
            onClick: () => openAnnouncementModal(),
          },
        ];
      default:
        return [
          {
            label: "Update jumpbridges",
            path: "/admin/jobs/jumpbridges/update",
            confirm: "Run jumpbridge validation and discovery update now?",
          },
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
  }, [
    group,
    standaloneAuth,
    openAllowedOrganizationsModal,
    openAnnouncementModal,
    standaloneTimers,
  ]);

  useEffect(() => {
    if (!standaloneTimers && group === "timer_data") {
      setGroup("map_data");
    }
  }, [group, standaloneTimers]);
  const panelActions = (
    <select
      className="select select-xs bg-base-300/70"
      value={group}
      onChange={(event) => setGroup(event.target.value as ActionGroupWithSite)}
    >
      {availableGroups.map(([key, label]) => (
        <option key={key} value={key}>
          {label}
        </option>
      ))}
    </select>
  );

  return (
    <Panel
      title="Admin Actions"
      hint="Trigger background jobs and updates."
      actions={panelActions}
      bodyClassName="space-y-4"
    >
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <button
            key={action.path ?? action.label}
            className="btn btn-xs btn-info btn-outline"
            onClick={() => {
              if (action.onClick) {
                action.onClick();
                return;
              }
              if (action.confirm) {
                if (!action.path) return;
                const actionPath = action.path;
                requestConfirm({
                  title: action.label,
                  body: action.confirm,
                  onConfirm: () => runAction(action.label, actionPath),
                  confirmLabel: "Run",
                  cancelLabel: "Cancel",
                  tone: "default",
                });
                return;
              }
              if (!action.path) return;
              void runAction(action.label, action.path);
            }}
            disabled={loadingLabel !== null}
          >
            {action.label}
          </button>
        ))}
      </div>
      {loadingLabel && (
        <p className="text-xs text-slate-400">Running: {loadingLabel}…</p>
      )}
    </Panel>
  );
}
