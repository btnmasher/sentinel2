import { useMemo, useState } from "react";
import { useUIStore } from "@/app/store/uiStore";
import { useAdminMapDataStore } from "../store/adminMapDataStore";

type ActionGroup = "map_data" | "characters" | "maintenance";

const GROUP_LABELS: Record<ActionGroup, string> = {
  map_data: "Map Data",
  characters: "Characters",
  maintenance: "Maintenance",
};

export default function JobActionsSection() {
  const requestConfirm = useUIStore((s) => s.requestConfirm);
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
          {
            label: "Build metro positions",
            path: "/admin/map-data/metro-positions",
          },
        ];
    }
  }, [group]);

  return (
    <section className="card bg-base-200/70 border border-slate-800">
      <div className="card-body space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="font-display text-2xl">Job Actions</h2>
            <p className="text-xs text-slate-400">
              Trigger background jobs and updates.
            </p>
          </div>
          <select
            className="select select-xs bg-base-300/70"
            value={group}
            onChange={(event) =>
              setGroup(event.target.value as ActionGroup)
            }
          >
            {Object.entries(GROUP_LABELS).map(([key, label]) => (
              <option key={key} value={key}>
                {label}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <button
              key={action.path}
              className="btn btn-xs btn-info btn-outline"
              onClick={() => {
                if (action.confirm) {
                  requestConfirm(action.label, action.confirm, () =>
                    runAction(action.label, action.path),
                  );
                  return;
                }
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
      </div>
    </section>
  );
}
